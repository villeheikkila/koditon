package web

import (
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"koditon-go/internal/domain/ads"
	"koditon-go/internal/platform/logging"
)

//go:embed templates/*.html
var templateFS embed.FS

var templates = template.Must(template.New("").Funcs(template.FuncMap{
	"formatPrice": formatPrice,
	"formatArea":  formatArea,
}).ParseFS(templateFS, "templates/*.html"))

type Handler struct {
	ads           *ads.Service
	shortcutBase  string
	frontdoorBase string
	logger        *slog.Logger
}

func NewHandler(adsService *ads.Service, shortcutBase, frontdoorBase string, logger *slog.Logger) *Handler {
	return &Handler{
		ads:           adsService,
		shortcutBase:  shortcutBase,
		frontdoorBase: frontdoorBase,
		logger:        logger,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /detail/{id...}", h.handleDetail)
	mux.HandleFunc("GET /", h.handleIndex)
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		logging.With(h.logger, logging.Op("web.index.render")).ErrorContext(r.Context(), "render index failed", "error", err, "outcome", logging.OutcomeError)
	}
}

func (h *Handler) handleDetail(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	if rawID == "" {
		http.NotFound(w, r)
		return
	}

	canonicalID, err := ads.ResolveInput(rawID, h.shortcutBase, h.frontdoorBase)
	if err != nil {
		h.renderError(w, http.StatusBadRequest, fmt.Sprintf("Invalid ID: %s", rawID))
		return
	}

	detail, err := h.ads.DetailByCanonicalID(r.Context(), canonicalID)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "not found") {
			h.renderError(w, http.StatusNotFound, "Entity not found")
			return
		}
		logging.With(h.logger, logging.Op("web.detail.lookup"), slog.String("canonical_id", canonicalID)).ErrorContext(r.Context(), "detail lookup failed", "error", err, "outcome", logging.OutcomeError)
		h.renderError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "detail.html", detail); err != nil {
		logging.With(h.logger, logging.Op("web.detail.render"), slog.String("canonical_id", canonicalID)).ErrorContext(r.Context(), "render detail failed", "error", err, "outcome", logging.OutcomeError)
	}
}

func (h *Handler) renderError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	data := struct {
		Status  int
		Message string
	}{status, message}
	if err := templates.ExecuteTemplate(w, "error.html", data); err != nil {
		http.Error(w, message, status)
	}
}

func formatPrice(p *int64) string {
	if p == nil {
		return "-"
	}
	v := *p
	s := fmt.Sprintf("%d", v)
	if v < 0 {
		s = fmt.Sprintf("%d", -v)
	}
	if len(s) <= 3 {
		if v < 0 {
			return fmt.Sprintf("-%s €", s)
		}
		return fmt.Sprintf("%s €", s)
	}
	var b strings.Builder
	offset := len(s) % 3
	if offset > 0 {
		b.WriteString(s[:offset])
	}
	for i := offset; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(s[i : i+3])
	}
	if v < 0 {
		return fmt.Sprintf("-%s €", b.String())
	}
	return fmt.Sprintf("%s €", b.String())
}

func formatArea(a *float64) string {
	if a == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f m²", *a)
}
