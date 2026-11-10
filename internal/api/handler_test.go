package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"searchtags/internal/tags"
)

// fakeStore — заглушка вместо ClickHouse. Запоминает, с какими запросами её вызывали.
type fakeStore struct {
	data  map[string][]tags.Tag
	err   error
	calls []string
}

func (f *fakeStore) Tags(_ context.Context, query string) ([]tags.Tag, error) {
	f.calls = append(f.calls, query)
	if f.err != nil {
		return nil, f.err
	}
	return f.data[query], nil // для неизвестного запроса вернётся nil
}

func newServer(store tags.Store) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mux := http.NewServeMux()
	NewHandler(store, logger).Register(mux)
	return mux
}

func get(h http.Handler, q string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/tags?q="+url.QueryEscape(q), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestTags_EmptyQuery_Returns400(t *testing.T) {
	// Нет параметра, пустой параметр и «одни пробелы» — всё это пустой запрос.
	targets := []string{"/api/tags", "/api/tags?q=", "/api/tags?q=%20%20%20"}

	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			store := &fakeStore{}
			h := newServer(store)

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("статус = %d, ожидали 400", rec.Code)
			}
			if len(store.calls) != 0 {
				t.Errorf("до хранилища не должны были дойти, а вызовов: %v", store.calls)
			}
		})
	}
}

func TestTags_ReturnsTagsInOrder(t *testing.T) {
	want := []tags.Tag{
		{Tag: "красное платье", Users: 9264},
		{Tag: "вечернее платье", Users: 6609},
	}
	h := newServer(&fakeStore{data: map[string][]tags.Tag{"платье": want}})

	rec := get(h, "Платье ")

	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидали 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, ожидали application/json", ct)
	}
	var got []tags.Tag
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("ответ не JSON: %v", err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("теги = %v, ожидали %v", got, want)
	}
}

func TestTags_NormalizationGivesSameAnswer(t *testing.T) {
	store := &fakeStore{data: map[string][]tags.Tag{
		"платье": {{Tag: "красное платье", Users: 5}},
	}}
	h := newServer(store)

	variants := []string{"платье", "Платье ", "  ПЛАТЬЕ", "платье\t"}
	first := get(h, variants[0]).Body.String()

	for _, v := range variants {
		if body := get(h, v).Body.String(); body != first {
			t.Errorf("для %q ответ %q отличается от %q", v, body, first)
		}
	}
	for _, call := range store.calls {
		if call != "платье" {
			t.Errorf("в хранилище ушёл %q, ожидали нормализованное %q", call, "платье")
		}
	}
}

func TestTags_NoTags_ReturnsEmptyListWith200(t *testing.T) {
	h := newServer(&fakeStore{}) // хранилище возвращает nil

	rec := get(h, "неизвестный запрос")

	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидали 200", rec.Code)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Errorf("тело = %q, ожидали []", body)
	}
}

func TestTags_StoreError_Returns500WithoutDetails(t *testing.T) {
	h := newServer(&fakeStore{err: errors.New("secret connection string")})

	rec := get(h, "платье")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, ожидали 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("в ответ утекли подробности ошибки: %s", rec.Body.String())
	}
}

func TestTags_WrongMethod_Returns405(t *testing.T) {
	h := newServer(&fakeStore{})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/tags?q=платье", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("статус = %d, ожидали 405", rec.Code)
	}
}
