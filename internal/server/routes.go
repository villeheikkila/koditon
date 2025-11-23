package server

import (
	"encoding/json"
	"koditon-go/internal/config"
	"log"
	"net/http"
)

func addRoutes(mux *http.ServeMux, logger *log.Logger, cfg config.Config) {
	mux.Handle("/healthz", handleHealth())
	mux.Handle("/api/v1/ping", handlePing(logger))
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

func encode[T any](w http.ResponseWriter, r *http.Request, status int, v T) error {
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
