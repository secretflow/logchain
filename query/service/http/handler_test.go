package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetStatusByRequestID_PathTraversal(t *testing.T) {
	// Create handler with nil service since we're only testing path validation
	handler := &Handler{service: nil}

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "valid request ID",
			path:           "/v1/query/status/123e4567-e89b-12d3-a456-426614174000",
			expectedStatus: http.StatusUnauthorized, // Will fail auth, but passes validation
			description:    "normal UUID should pass validation",
		},
		{
			name:           "path traversal with ..",
			path:           "/v1/query/status/../../../etc/passwd",
			expectedStatus: http.StatusBadRequest,
			description:    "should reject path traversal with ..",
		},
		{
			name:           "path with forward slash",
			path:           "/v1/query/status/abc/def",
			expectedStatus: http.StatusBadRequest,
			description:    "should reject path with /",
		},
		{
			name:           "path traversal combined",
			path:           "/v1/query/status/../../etc/passwd",
			expectedStatus: http.StatusBadRequest,
			description:    "should reject combined path traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			handler.GetStatusByRequestID(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d", tt.description, tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestAuditLogByHash_PathTraversal(t *testing.T) {
	// Create handler with nil service since we're only testing path validation
	handler := &Handler{service: nil}

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "valid log hash",
			path:           "/v1/audit/log/abcdef1234567890",
			expectedStatus: http.StatusUnauthorized, // Will fail auth, but passes validation
			description:    "normal hash should pass validation",
		},
		{
			name:           "path traversal with ..",
			path:           "/v1/audit/log/../../../etc/passwd",
			expectedStatus: http.StatusBadRequest,
			description:    "should reject path traversal with ..",
		},
		{
			name:           "path with forward slash",
			path:           "/v1/audit/log/abc/def",
			expectedStatus: http.StatusBadRequest,
			description:    "should reject path with /",
		},
		{
			name:           "path traversal combined",
			path:           "/v1/audit/log/../../etc/passwd",
			expectedStatus: http.StatusBadRequest,
			description:    "should reject combined path traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			handler.AuditLogByHash(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d", tt.description, tt.expectedStatus, w.Code)
			}
		})
	}
}
