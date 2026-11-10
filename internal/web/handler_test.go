package web

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"searchtags/internal/tags"
)

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
	return f.data[query], nil
}

func newServer(t *testing.T, store tags.Store) http.Handler {
	t.Helper()
	h, err := NewHandler(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	mux := http.NewServeMux()
	h.Register(mux)
	return mux
}

func get(h http.Handler, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestPage_EscapesUserInput(t *testing.T) {
	h := newServer(t, &fakeStore{})
	evil := `"><script>alert(1)</script>`

	rec := get(h, "/?q="+url.QueryEscape(evil))
	body := rec.Body.String()

	if strings.Contains(body, "<script>") {
		t.Errorf("ввод пользователя попал на страницу неэкранированным")
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Errorf("ожидали экранированный вариант в теле страницы")
	}
}

func TestPage_NoTags_ShowsMessage(t *testing.T) {
	h := newServer(t, &fakeStore{})

	body := get(h, "/?q="+url.QueryEscape("абракадабра")).Body.String()

	if !strings.Contains(body, "Для этого запроса пока нет тегов") {
		t.Errorf("нет сообщения про отсутствие тегов")
	}
}

func TestPage_ShowsTagsAsLinks(t *testing.T) {
	h := newServer(t, &fakeStore{data: map[string][]tags.Tag{
		"платье": {{Tag: "красное платье", Users: 9264}},
	}})

	body := get(h, "/?q="+url.QueryEscape("Платье ")).Body.String()

	if !strings.Contains(body, `href="/?q=`) || !strings.Contains(body, ">красное платье</a>") {
		t.Errorf("тег не показан как ссылка, тело:\n%s", body)
	}
	if strings.Contains(body, "Для этого запроса пока нет тегов") {
		t.Errorf("сообщение «нет тегов» не должно показываться, когда теги есть")
	}
}

func TestPage_EmptyQuery_ShowsOnlyForm(t *testing.T) {
	store := &fakeStore{}
	h := newServer(t, store)

	rec := get(h, "/?q=")

	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидали 200", rec.Code)
	}
	if len(store.calls) != 0 {
		t.Errorf("при пустом запросе в базу ходить не нужно, вызовы: %v", store.calls)
	}
	if strings.Contains(rec.Body.String(), "пока нет тегов") {
		t.Errorf("на пустой форме не должно быть сообщения про теги")
	}
}

func TestPage_StoreError_Shows500(t *testing.T) {
	h := newServer(t, &fakeStore{err: errors.New("boom")})

	rec := get(h, "/?q="+url.QueryEscape("платье"))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("статус = %d, ожидали 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("подробности ошибки не должны попадать на страницу")
	}
}

func TestPage_UnknownPath_Returns404(t *testing.T) {
	h := newServer(t, &fakeStore{})

	if rec := get(h, "/qwerty"); rec.Code != http.StatusNotFound {
		t.Errorf("статус = %d, ожидали 404", rec.Code)
	}
}
