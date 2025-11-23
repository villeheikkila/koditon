package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"koditon-go/internal/config"
	"koditon-go/internal/db"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

func addRoutes(mux *http.ServeMux, logger *log.Logger, cfg config.Config, queries *db.Queries) {
	mux.Handle("/healthz", handleHealth())
	mux.Handle("/api/v1/ping", handlePing(logger))
	mux.Handle("/api/v1/cities", handleListCities(logger, queries))
	mux.Handle("/api/v1/transactions", handleListTransactions(logger, queries))
	mux.Handle("/", http.NotFoundHandler())
}

func handleHealth() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = encode(w, r, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})
}

func handlePing(logger *log.Logger) http.Handler {
	type request struct {
		Message string `json:"message"`
	}
	type response struct {
		Echo string `json:"echo"`
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, err := decode[request](r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		logger.Printf("ping received: %s", payload.Message)
		if err := encode(w, r, http.StatusOK, response{Echo: payload.Message}); err != nil {
			logger.Printf("encode response: %v", err)
		}
	})
}

func handleListCities(logger *log.Logger, queries *db.Queries) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		cities, err := queries.ListCitiesWithNeighborhoods(r.Context())
		if err != nil {
			logger.Printf("list cities with neighborhoods: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		response := mapCitiesWithNeighborhoods(cities)
		if err := encode(w, r, http.StatusOK, response); err != nil {
			logger.Printf("encode response: %v", err)
		}
	})
}

func handleListTransactions(logger *log.Logger, queries *db.Queries) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
					logger.Printf("invalid UUID %q: %v", uuidStr, err)
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

		transactions, err := queries.ListTransactionsByNeighborhoods(r.Context(), neighborhoodIDs)
		if err != nil {
			logger.Printf("list transactions by neighborhoods: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		response := make([]HintatiedotResponse, len(transactions))
		for i, row := range transactions {
			response[i] = mapTransactionResponse(row)
		}
		if err := encode(w, r, http.StatusOK, response); err != nil {
			logger.Printf("encode response: %v", err)
		}
	})
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
