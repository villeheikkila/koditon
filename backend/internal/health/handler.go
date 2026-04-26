package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"koditon-go/internal/buildinfo"
	"koditon-go/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool *pgxpool.Pool
	mode config.AppMode
	info buildinfo.Info
}

func New(pool *pgxpool.Pool, mode config.AppMode, info buildinfo.Info) *Handler {
	return &Handler{
		pool: pool,
		mode: mode,
		info: info,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/livez", h.handleLivez)
	mux.HandleFunc("/readyz", h.handleReadyz)
}

func (h *Handler) handleLivez(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	body := map[string]any{
		"status":     "ok",
		"service":    "koditon",
		"version":    h.info.Version,
		"commit":     h.info.Commit,
		"build_time": h.info.BuildTime,
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Handler) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	checkCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	checks := map[string]string{"database": "ok"}
	status := http.StatusOK
	if err := h.pool.Ping(checkCtx); err != nil {
		status = http.StatusServiceUnavailable
		checks["database"] = "unavailable"
	}
	body := map[string]any{
		"status": statusText(status),
		"mode":   h.mode.String(),
		"checks": checks,
	}
	writeJSON(w, status, body)
}

func statusText(status int) string {
	if status >= 200 && status < 300 {
		return "ok"
	}
	return "degraded"
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
