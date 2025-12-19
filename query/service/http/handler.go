package http

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"tlng/query/auth"
	"tlng/query/service/core"
)

// Handler wraps the query service with HTTP handlers
type Handler struct {
	service *core.Service
	logger  *log.Logger
}

// NewHandler creates a new HTTP handler
func NewHandler(service *core.Service, logger *log.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers all query API routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// API 1: Query by request_id (API Key auth)
	mux.Handle("/v1/query/status/", auth.RequireAPIKey(http.HandlerFunc(h.GetStatusByRequestID)))

	// API 2: Query by log content (API Key auth)
	mux.Handle("/v1/query_by_content", auth.RequireAPIKey(http.HandlerFunc(h.QueryByContent)))

	// API 3: Audit log by hash (mTLS auth)
	mux.Handle("/v1/audit/log/", auth.RequireMTLS(http.HandlerFunc(h.AuditLogByHash)))
}

// GetStatusByRequestID handles GET /v1/query/status/{request_id}
func (h *Handler) GetStatusByRequestID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract request_id from path
	path := strings.TrimPrefix(r.URL.Path, "/v1/query/status/")
	requestID := strings.TrimSpace(path)
	if requestID == "" {
		h.writeError(w, http.StatusBadRequest, "missing request_id")
		return
	}

	// Validate request_id to prevent path traversal
	if strings.Contains(requestID, "..") || strings.Contains(requestID, "/") {
		h.writeError(w, http.StatusBadRequest, "invalid request_id: path traversal characters not allowed")
		return
	}

	// Extract auth context
	authCtx := auth.ExtractAuthContext(r)
	if authCtx == nil || authCtx.OrgID == "" {
		h.writeError(w, http.StatusUnauthorized, "missing authentication context")
		return
	}

	// Call service
	result, err := h.service.GetStatusByRequestID(r.Context(), requestID, authCtx.OrgID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

// QueryByContentRequest represents the request body for content query
type QueryByContentRequest struct {
	LogContent string `json:"log_content"`
}

// QueryByContent handles POST /v1/query_by_content
func (h *Handler) QueryByContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Ensure the request body is closed when we're done
	defer r.Body.Close()

	// Parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req QueryByContentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if strings.TrimSpace(req.LogContent) == "" {
		h.writeError(w, http.StatusBadRequest, "log_content is required")
		return
	}

	// Extract auth context
	authCtx := auth.ExtractAuthContext(r)
	if authCtx == nil || authCtx.OrgID == "" {
		h.writeError(w, http.StatusUnauthorized, "missing authentication context")
		return
	}

	// Call service
	result, err := h.service.QueryByContent(r.Context(), req.LogContent, authCtx.OrgID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

// AuditLogByHash handles GET /v1/audit/log/{log_hash}
func (h *Handler) AuditLogByHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract log_hash from path
	path := strings.TrimPrefix(r.URL.Path, "/v1/audit/log/")
	logHash := strings.TrimSpace(path)
	if logHash == "" {
		h.writeError(w, http.StatusBadRequest, "missing log_hash")
		return
	}

	// Validate log_hash to prevent path traversal
	if strings.Contains(logHash, "..") || strings.Contains(logHash, "/") {
		h.writeError(w, http.StatusBadRequest, "invalid log_hash: path traversal characters not allowed")
		return
	}

	// Extract auth context (mTLS, member_id required)
	authCtx := auth.ExtractAuthContext(r)
	if authCtx == nil {
		h.writeError(w, http.StatusUnauthorized, "missing authentication context")
		return
	}

	if authCtx.MemberID == "" {
		h.writeError(w, http.StatusForbidden, "member_id required for audit API")
		return
	}

	// Call service (no org restriction for consortium members)
	result, err := h.service.AuditLogByHash(r.Context(), logHash)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// writeError writes a JSON error response
func (h *Handler) writeError(w http.ResponseWriter, statusCode int, message string) {
	h.writeJSON(w, statusCode, ErrorResponse{Error: message})
}

// writeJSON writes a JSON response
func (h *Handler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Status code already sent, can only log the error
		h.logger.Printf("ERROR: Failed to encode JSON response: %v", err)
	}
}

// handleServiceError maps service errors to HTTP status codes using typed error checking
func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, core.ErrLogNotFound):
		h.writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, core.ErrPermissionDenied):
		h.writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, core.ErrInvalidRequest):
		h.writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, core.ErrBlockchainError):
		h.writeError(w, http.StatusInternalServerError, err.Error())
	default:
		h.writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
