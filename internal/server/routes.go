package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Server) addRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/v1/ping", s.handlePing)
	mux.HandleFunc("/api/v1/cities", s.handleListCities)
	mux.HandleFunc("/api/v1/transactions", s.handleListTransactions)
	mux.HandleFunc("/api/v1/hintatiedot/cities", s.handleFetchHintatiedotCities)
	mux.Handle("/", http.NotFoundHandler())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = encode(w, r, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) handleListCities(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse neighborhood UUIDs from query parameter
	neighborhoodIDs := []pgtype.UUID{}
	if v := r.URL.Query().Get("neighborhoods"); v != "" {
		uuidStrings := strings.Split(v, ",")
		for _, uuidStr := range uuidStrings {
			uuidStr = strings.TrimSpace(uuidStr)
			if uuidStr == "" {
				continue
			}
			parsedUUID, err := parseUUID(uuidStr)
			if err != nil {
				s.logger.WarnContext(r.Context(), "invalid neighborhood UUID", "uuid", uuidStr, "err", err)
				http.Error(w, "Invalid neighborhood UUID", http.StatusBadRequest)
				return
			}
			neighborhoodIDs = append(neighborhoodIDs, parsedUUID)
		}
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

func (s *Server) handleFetchHintatiedotCities(w http.ResponseWriter, r *http.Request) {
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
