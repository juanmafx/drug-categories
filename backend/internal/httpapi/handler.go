package httpapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/juanmafx/drug-categories/backend/internal/graphqlapi"
)

const (
	maxBodyBytes    = 16 << 10
	maxQueryBytes   = 8 << 10
	requestsPerMin  = 120
	rateLimitWindow = time.Minute
)

type Options struct {
	AllowedOrigins []string
	WriteToken     string
	Logger         *slog.Logger
	Now            func() time.Time
}

type graphQLRequest struct {
	Query         string         `json:"query"`
	OperationName string         `json:"operationName"`
	Variables     map[string]any `json:"variables"`
}

type server struct {
	service        *graphqlapi.Service
	allowedOrigins []string
	writeTokenHash [sha256.Size]byte
	requiresToken  bool
	limiter        *rateLimiter
	logger         *slog.Logger
}

func NewHandler(service *graphqlapi.Service, options Options) http.Handler {
	now := options.Now
	if now == nil {
		now = time.Now
	}
	api := &server{
		service:        service,
		allowedOrigins: slices.Clone(options.AllowedOrigins),
		writeTokenHash: sha256.Sum256([]byte(options.WriteToken)),
		requiresToken:  options.WriteToken != "",
		limiter:        newRateLimiter(requestsPerMin, rateLimitWindow, now),
		logger:         options.Logger,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", api.handleGraphQL)
	mux.HandleFunc("/healthz", handleHealth)
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := w.Header()
		headers.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("Referrer-Policy", "no-referrer")
		headers.Set("Cache-Control", "no-store")
		headers.Set("Cross-Origin-Resource-Policy", "same-site")
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Vary", "Origin")
	origin := r.Header.Get("Origin")
	if origin != "" {
		if !slices.Contains(s.allowedOrigins, origin) {
			writeError(w, http.StatusForbidden, "origin not allowed")
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	switch r.Method {
	case http.MethodOptions:
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "600")
		w.WriteHeader(http.StatusNoContent)
		return
	case http.MethodPost:
	default:
		w.Header().Set("Allow", "POST, OPTIONS")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !s.limiter.allow(r.RemoteAddr) {
		w.Header().Set("Retry-After", "60")
		writeError(w, http.StatusTooManyRequests, "too many requests")
		return
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return
	}

	request, status, message := decodeRequest(w, r)
	if status != 0 {
		writeError(w, status, message)
		return
	}

	response, err := s.service.Execute(r.Context(), graphqlapi.Request{
		Query:         request.Query,
		OperationName: request.OperationName,
		Variables:     request.Variables,
		CanWrite:      s.canWrite(r),
	})
	switch {
	case errors.Is(err, graphqlapi.ErrUnauthorized):
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeJSON(w, http.StatusUnauthorized, response)
	case errors.Is(err, graphqlapi.ErrRejected):
		writeJSON(w, http.StatusBadRequest, response)
	case err != nil:
		s.logger.Error("graphql request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	default:
		writeJSON(w, http.StatusOK, response)
	}
}

func decodeRequest(w http.ResponseWriter, r *http.Request) (graphQLRequest, int, string) {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	var request graphQLRequest
	if err := decoder.Decode(&request); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return request, http.StatusRequestEntityTooLarge, "request body too large"
		}
		return request, http.StatusBadRequest, "invalid json"
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return request, http.StatusBadRequest, "invalid json"
	}
	if strings.TrimSpace(request.Query) == "" {
		return request, http.StatusBadRequest, "query is required"
	}
	if len(request.Query) > maxQueryBytes {
		return request, http.StatusRequestEntityTooLarge, "query too large"
	}
	return request, 0, ""
}

func (s *server) canWrite(r *http.Request) bool {
	if !s.requiresToken {
		return true
	}
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return false
	}
	presented := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return subtle.ConstantTimeCompare(presented[:], s.writeTokenHash[:]) == 1
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, graphqlapi.Response{Errors: []graphqlapi.Error{{Message: message}}})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
