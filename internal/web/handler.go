package web

import (
	"bytes"
	"embed"
	"html/template"
	"log/slog"
	"net/http"

	"searchtags/internal/tags"
)

//go:embed templates/index.html
var templatesFS embed.FS

// pageData — всё, что шаблон index.html может показать.
type pageData struct {
	Query    string     // текст, который ввёл пользователь (возвращаем в поле как есть)
	Searched bool       // был ли поиск (если нет, показываем только форму)
	Tags     []tags.Tag // найденные теги
	Error    string     // сообщение об ошибке, если не удалось загрузить теги
}

type Handler struct {
	store  tags.Store
	logger *slog.Logger
	tmpl   *template.Template
}

func NewHandler(store tags.Store, logger *slog.Logger) (*Handler, error) {
	// Шаблон разбирается один раз при старте. Если в нём опечатка, сервис не запустится.
	tmpl, err := template.ParseFS(templatesFS, "templates/index.html")
	if err != nil {
		return nil, err
	}
	return &Handler{store: store, logger: logger, tmpl: tmpl}, nil
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/", h.handleIndex)
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Шаблон "/" в ServeMux ловит вообще все адреса, которые не нашли другого обработчика.
	// Поэтому лишние пути вроде /abc отдаём как 404 вручную.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	raw := r.URL.Query().Get("q")
	data := pageData{Query: raw}
	status := http.StatusOK

	// Нормализуем так же, как API. Если после нормализации пусто (поле не заполнено
	// или одни пробелы), просто показываем форму без поиска.
	if q := tags.Normalize(raw); q != "" {
		data.Searched = true
		result, err := h.store.Tags(r.Context(), q)
		if err != nil {
			h.logger.Error("load tags for page failed", "query", q, "error", err)
			data.Error = "Не удалось загрузить теги. Попробуйте ещё раз позже."
			status = http.StatusInternalServerError
		} else {
			data.Tags = result
		}
	}

	h.render(w, status, data)
}

// render сначала собирает страницу в памяти и только потом отправляет клиенту.
// Если шаблон упадёт посередине, клиент получит чистую ошибку, а не обрезанную страницу с кодом 200.
func (h *Handler) render(w http.ResponseWriter, status int, data pageData) {
	var buf bytes.Buffer
	if err := h.tmpl.Execute(&buf, data); err != nil {
		h.logger.Error("render page failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}
