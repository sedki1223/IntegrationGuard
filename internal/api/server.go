package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"integration-guard/internal/batch"
	"integration-guard/internal/config"
	"integration-guard/internal/store"
	"integration-guard/internal/validator"
)

type Repository interface {
	ListIntegrations(ctx context.Context) ([]store.Integration, error)
	CreateIntegration(ctx context.Context, cfg config.Config) (store.Integration, error)
	GetIntegration(ctx context.Context, id string) (store.Integration, error)
	UpdateIntegration(ctx context.Context, id string, cfg config.Config) (store.Integration, error)
	DeleteIntegration(ctx context.Context, id string) error
	CreateRun(ctx context.Context, integration store.Integration, status string, report any) (store.Run, error)
	ListRuns(ctx context.Context, limit int) ([]store.Run, error)
	GetRun(ctx context.Context, id string) (store.Run, error)
}

type Server struct {
	repository Repository
	mux        *http.ServeMux
}

func New(repository Repository) http.Handler {
	s := &Server{repository: repository, mux: http.NewServeMux()}
	s.routes()
	return s.recoverMiddleware(s.corsMiddleware(s.mux))
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.health)
	s.mux.HandleFunc("GET /api/v1/integrations", s.listIntegrations)
	s.mux.HandleFunc("POST /api/v1/integrations", s.createIntegration)
	s.mux.HandleFunc("GET /api/v1/integrations/{id}", s.getIntegration)
	s.mux.HandleFunc("PUT /api/v1/integrations/{id}", s.updateIntegration)
	s.mux.HandleFunc("DELETE /api/v1/integrations/{id}", s.deleteIntegration)
	s.mux.HandleFunc("POST /api/v1/integrations/{id}/runs", s.runValidation)
	s.mux.HandleFunc("GET /api/v1/runs", s.listRuns)
	s.mux.HandleFunc("GET /api/v1/runs/{id}", s.getRun)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "integration-guard-api"})
}

func (s *Server) listIntegrations(w http.ResponseWriter, r *http.Request) {
	items, err := s.repository.ListIntegrations(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createIntegration(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := decodeJSON(w, r, &cfg); err != nil {
		badRequest(w, err.Error())
		return
	}
	if err := cfg.Validate(); err != nil {
		badRequest(w, err.Error())
		return
	}
	item, err := s.repository.CreateIntegration(r.Context(), cfg)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) getIntegration(w http.ResponseWriter, r *http.Request) {
	item, err := s.repository.GetIntegration(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) updateIntegration(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := decodeJSON(w, r, &cfg); err != nil {
		badRequest(w, err.Error())
		return
	}
	if err := cfg.Validate(); err != nil {
		badRequest(w, err.Error())
		return
	}
	item, err := s.repository.UpdateIntegration(r.Context(), r.PathValue("id"), cfg)
	if errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) deleteIntegration(w http.ResponseWriter, r *http.Request) {
	err := s.repository.DeleteIntegration(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return
	}
	if err != nil {
		// Runs reference integrations and are intentionally retained as audit history.
		writeJSON(w, http.StatusConflict, map[string]string{"error": "integration has validation history or could not be deleted"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type validateRequest struct {
	Source      json.RawMessage `json:"source"`
	Destination json.RawMessage `json:"destination"`
}

type reportEnvelope struct {
	IntegrationID   string `json:"integration_id"`
	IntegrationName string `json:"integration_name"`
	Valid           bool   `json:"valid"`
	Result          any    `json:"result"`
}

func (s *Server) runValidation(w http.ResponseWriter, r *http.Request) {
	integration, err := s.repository.GetIntegration(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}

	var input validateRequest
	if err := decodeJSON(w, r, &input); err != nil {
		badRequest(w, err.Error())
		return
	}
	source, err := decodeRecords(input.Source)
	if err != nil {
		badRequest(w, "source: "+err.Error())
		return
	}
	destination, err := decodeRecords(input.Destination)
	if err != nil {
		badRequest(w, "destination: "+err.Error())
		return
	}

	var result any
	var valid bool
	cfg := integration.Config
	if cfg.MatchKey.SourceField != "" || cfg.MatchKey.DestinationField != "" {
		batchResult := batch.Compare(source, destination, cfg.MatchKey, cfg.Rules)
		result, valid = batchResult, batchResult.Valid
	} else {
		if len(source) != 1 || len(destination) != 1 {
			badRequest(w, "single-record configurations require exactly one source and one destination object")
			return
		}
		single := validator.Compare(source[0], destination[0], cfg.Rules)
		result, valid = single, single.Valid
	}
	report := reportEnvelope{
		IntegrationID:   integration.ID,
		IntegrationName: integration.Name,
		Valid:           valid,
		Result:          result,
	}
	status := "passed"
	if !valid {
		status = "failed"
	}
	run, err := s.repository.CreateRun(r.Context(), integration, status, report)
	if err != nil {
		serverError(w, err)
		return
	}
	code := http.StatusOK
	if !valid {
		code = http.StatusUnprocessableEntity
	}
	writeJSON(w, code, map[string]any{"run": run, "report": report})
}

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			badRequest(w, "limit must be between 1 and 200")
			return
		}
		limit = parsed
	}
	items, err := s.repository.ListRuns(r.Context(), limit)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	item, err := s.repository.GetRun(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("request must contain exactly one JSON value")
		}
		return fmt.Errorf("invalid trailing JSON: %w", err)
	}
	return nil
}

func decodeRecords(raw json.RawMessage) ([]validator.Record, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, errors.New("field is required")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	trimmed := bytes.TrimSpace(raw)
	switch trimmed[0] {
	case '{':
		var record validator.Record
		if err := decoder.Decode(&record); err != nil {
			return nil, err
		}
		return []validator.Record{record}, nil
	case '[':
		var records []validator.Record
		if err := decoder.Decode(&records); err != nil {
			return nil, err
		}
		if records == nil {
			return []validator.Record{}, nil
		}
		return records, nil
	default:
		return nil, errors.New("must be a JSON object or array of objects")
	}
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := strings.TrimSpace(getenv("CORS_ORIGIN", "http://localhost:5173"))
		if origin != "" && origin == allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic handling %s %s: %v", r.Method, r.URL.Path, recovered)
				serverError(w, errors.New("internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if status != http.StatusNoContent {
		if err := json.NewEncoder(w).Encode(value); err != nil {
			log.Printf("encode response: %v", err)
		}
	}
}

func badRequest(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": message})
}

func notFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "resource not found"})
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("API error: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
