package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

var _ ServerInterface = (*Server)(nil)

func (s *Server) addRoutes(mux *http.ServeMux) {
	HandlerFromMux(s, mux)
	mux.Handle("/", http.NotFoundHandler())
}

func (s *Server) Healthz(w http.ResponseWriter, r *http.Request) {
	_ = encode(w, r, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (s *Server) Ping(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Message string `json:"message"`
	}
	type response struct {
		Echo string `json:"echo"`
	}

	payload, err := decode[request](r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.logger.InfoContext(r.Context(), "ping received", "message", payload.Message)
	if err := encode(w, r, http.StatusOK, response{Echo: payload.Message}); err != nil {
		s.logger.ErrorContext(r.Context(), "encode response", "err", err)
	}
}

func (s *Server) FetchTransactions(w http.ResponseWriter, r *http.Request) {
	transactions, err := s.hintatiedotAPI.GetAllTransactions(r.Context(), "Helsinki")
	if err != nil {
		s.logger.ErrorContext(r.Context(), "fetch all transactions from hintatiedot", "err", err)
		http.Error(w, "Failed to fetch transactions", http.StatusInternalServerError)
		return
	}
	s.logger.InfoContext(r.Context(), "list of transactions fetched", "count", len(transactions))
}

func (s *Server) ListCities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cities, err := s.db.ListCitiesWithNeighborhoods(r.Context())
	if err != nil {
		s.logger.ErrorContext(r.Context(), "list cities with neighborhoods", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	response := mapCitiesWithNeighborhoods(cities)
	if err := encode(w, r, http.StatusOK, response); err != nil {
		s.logger.ErrorContext(r.Context(), "encode response", "err", err)
	}
}

func (s *Server) ListTransactions(w http.ResponseWriter, r *http.Request, params ListTransactionsParams) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	neighborhoodIDs := make([]pgtype.UUID, len(params.Neighborhoods))
	for i, neighborhoodID := range params.Neighborhoods {
		parsedUUID, err := parseUUID(neighborhoodID.String())
		if err != nil {
			s.logger.WarnContext(r.Context(), "invalid neighborhood UUID", "uuid", neighborhoodID.String(), "err", err)
			http.Error(w, "Invalid neighborhood UUID", http.StatusBadRequest)
			return
		}
		neighborhoodIDs[i] = parsedUUID
	}

	if len(neighborhoodIDs) == 0 {
		http.Error(w, "At least one neighborhood UUID is required", http.StatusBadRequest)
		return
	}

	transactions, err := s.db.ListTransactionsByNeighborhoods(r.Context(), neighborhoodIDs)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "list transactions by neighborhoods", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]HintatiedotResponse, len(transactions))
	for i, row := range transactions {
		response[i] = mapTransactionResponse(row)
	}
	if err := encode(w, r, http.StatusOK, response); err != nil {
		s.logger.ErrorContext(r.Context(), "encode response", "err", err)
	}
}

func (s *Server) FetchHintatiedotCities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cities, err := s.hintatiedotAPI.FetchCities(r.Context())
	if err != nil {
		s.logger.ErrorContext(r.Context(), "fetch cities from hintatiedot", "err", err)
		http.Error(w, "Failed to fetch cities", http.StatusInternalServerError)
		return
	}

	if err := encode(w, r, http.StatusOK, map[string][]string{"cities": cities}); err != nil {
		s.logger.ErrorContext(r.Context(), "encode response", "err", err)
	}
}

func (s *Server) SyncHintatiedotCity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type request struct {
		City string `json:"city"`
	}
	type response struct {
		Status string `json:"status"`
		City   string `json:"city"`
	}

	payload, err := decode[request](r)
	if err != nil {
		s.logger.WarnContext(r.Context(), "invalid sync city payload", "err", err, "content_type", r.Header.Get("Content-Type"))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	city := strings.TrimSpace(payload.City)
	if city == "" {
		s.logger.WarnContext(r.Context(), "sync city request missing city name")
		http.Error(w, "City is required", http.StatusBadRequest)
		return
	}

	s.logger.InfoContext(r.Context(), "initiating city sync", "city", city)

	if err := s.hintatiedotSync.SyncCity(r.Context(), city); err != nil {
		s.logger.ErrorContext(r.Context(), "city sync failed", "city", city, "err", err)
		http.Error(w, fmt.Sprintf("Failed to sync city: %v", err), http.StatusInternalServerError)
		return
	}

	s.logger.InfoContext(r.Context(), "city sync succeeded", "city", city)

	if err := encode(w, r, http.StatusOK, response{Status: "ok", City: city}); err != nil {
		s.logger.ErrorContext(r.Context(), "failed to encode sync city response", "city", city, "err", err)
	}
}

func encode[T any](w http.ResponseWriter, _ *http.Request, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func decode[T any](r *http.Request) (T, error) {
	var v T
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&v); err != nil {
		return v, err
	}
	return v, nil
}

func parseUUID(s string) (pgtype.UUID, error) {
	// Remove hyphens from UUID string
	s = strings.ReplaceAll(s, "-", "")

	if len(s) != 32 {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID length: %d", len(s))
	}

	bytes, err := hex.DecodeString(s)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID format: %w", err)
	}

	var uuidBytes [16]byte
	copy(uuidBytes[:], bytes)

	return pgtype.UUID{
		Bytes: uuidBytes,
		Valid: true,
	}, nil
}
