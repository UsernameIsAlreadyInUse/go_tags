package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"searchtags/internal/tags"
)

type Handler struct {
	store  tags.Store
	logger *slog.Logger
}

func NewHandler(store tags.Store, logger *slog.Logger) *Handler {
	return &Handler{store: store, logger: logger}
}

// Register подключает маршруты обработчика к мультиплексору.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/tags", h.handleTags)
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) handleTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		h.writeJSON(w, http.StatusMethodNotAllowed, errorResponse{"method not allowed"})
		return
	}

	q := tags.Normalize(r.URL.Query().Get("q"))
	if q == "" {
		h.writeJSON(w, http.StatusBadRequest, errorResponse{"query parameter q is required"})
		return
	}

	result, err := h.store.Tags(r.Context(), q)
	if err != nil {
		h.logger.Error("load tags failed", "query", q, "error", err)
		h.writeJSON(w, http.StatusInternalServerError, errorResponse{"internal error"})
		return
	}

	// Гарантируем [] вместо null, какой бы Store ни использовался.
	if result == nil {
		result = []tags.Tag{}
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error("encode response failed", "error", err)
	}
}
