package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"tlng/query/auth"
	"tlng/query/service/core"
)

// Handler wraps the query service with HTTP handlers
type Handler struct {
	service *core.Service
}

// NewHandler creates a new HTTP handler
func NewHandler(service *core.Service) *Handler {
	return &Handler{service: service}
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
		writeError(w, http.StatusBadRequest, "missing request_id")
		return
	}

	// Validate request_id to prevent path traversal
	if strings.Contains(requestID, "..") || strings.Contains(requestID, "/") {
		writeError(w, http.StatusBadRequest, "invalid request_id: path traversal characters not allowed")
		return
	}

	// Extract auth context
	authCtx := auth.ExtractAuthContext(r)
	if authCtx == nil || authCtx.OrgID == "" {
		writeError(w, http.StatusUnauthorized, "missing authentication context")
		return
	}

	// Call service
	result, err := h.service.GetStatusByRequestID(r.Context(), requestID, authCtx.OrgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
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

	// Parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	var req QueryByContentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if strings.TrimSpace(req.LogContent) == "" {
		writeError(w, http.StatusBadRequest, "log_content is required")
		return
	}

	// Extract auth context
	authCtx := auth.ExtractAuthContext(r)
	if authCtx == nil || authCtx.OrgID == "" {
		writeError(w, http.StatusUnauthorized, "missing authentication context")
		return
	}

	// Call service
	result, err := h.service.QueryByContent(r.Context(), req.LogContent, authCtx.OrgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
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
		writeError(w, http.StatusBadRequest, "missing log_hash")
		return
	}

	// Validate log_hash to prevent path traversal
	if strings.Contains(logHash, "..") || strings.Contains(logHash, "/") {
		writeError(w, http.StatusBadRequest, "invalid log_hash: path traversal characters not allowed")
		return
	}

	// Extract auth context (mTLS, member_id required)
	authCtx := auth.ExtractAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "missing authentication context")
		return
	}

	if authCtx.MemberID == "" {
		writeError(w, http.StatusForbidden, "member_id required for audit API")
		return
	}

	// Call service (no org restriction for consortium members)
	result, err := h.service.AuditLogByHash(r.Context(), logHash)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// writeError writes a JSON error response
func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, ErrorResponse{Error: message})
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// handleServiceError maps service errors to HTTP status codes
func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case strings.Contains(err.Error(), "not found"):
		writeError(w, http.StatusNotFound, err.Error())
	case strings.Contains(err.Error(), "unauthorized"):
		writeError(w, http.StatusForbidden, err.Error())
	case strings.Contains(err.Error(), "invalid"):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
