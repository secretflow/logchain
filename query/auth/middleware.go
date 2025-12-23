package auth

import (
	"context"
	"net/http"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const authContextKey contextKey = "auth_context"

// AuthContext holds the authentication information extracted from HTTP headers
type AuthContext struct {
	ClientID    string // from X-API-Client-ID
	OrgID       string // from X-Client-Org-ID
	AuthMethod  string // "api-key" or "mtls"
	CertSubject string // from X-Cert-Subject (mTLS only)
	MemberID    string // from X-Member-ID (mTLS only)
}

// ExtractAuthContext extracts authentication information from HTTP headers
// This assumes the request has already been authenticated by Nginx
func ExtractAuthContext(r *http.Request) *AuthContext {
	authMethod := r.Header.Get("X-Auth-Method")
	if authMethod == "" {
		return nil
	}

	ctx := &AuthContext{
		AuthMethod: authMethod,
	}

	// Extract common headers
	if authMethod == "api-key" {
		ctx.ClientID = r.Header.Get("X-API-Client-ID")
		ctx.OrgID = r.Header.Get("X-Client-Org-ID")
	} else if authMethod == "mtls" {
		ctx.CertSubject = r.Header.Get("X-Cert-Subject")
		ctx.MemberID = r.Header.Get("X-Member-ID")
	}

	return ctx
}

// GetAuthContext retrieves the AuthContext from the request context
func GetAuthContext(ctx context.Context) *AuthContext {
	authCtx, ok := ctx.Value(authContextKey).(*AuthContext)
	if !ok {
		return nil
	}
	return authCtx
}

// WithAuthContext adds the AuthContext to the request context
func WithAuthContext(ctx context.Context, authCtx *AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, authCtx)
}

// RequireAPIKey is a middleware that requires API Key authentication
func RequireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := ExtractAuthContext(r)
		if authCtx == nil || authCtx.AuthMethod != "api-key" {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Unauthorized","message":"API key authentication required"}`, http.StatusUnauthorized)
			return
		}

		// Validate that required fields are present
		if authCtx.ClientID == "" {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Unauthorized","message":"Missing client ID"}`, http.StatusUnauthorized)
			return
		}
		if authCtx.OrgID == "" {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Unauthorized","message":"Missing organization ID"}`, http.StatusUnauthorized)
			return
		}

		// Add auth context to request context
		ctx := WithAuthContext(r.Context(), authCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireMTLS is a middleware that requires mTLS authentication
func RequireMTLS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := ExtractAuthContext(r)
		if authCtx == nil || authCtx.AuthMethod != "mtls" {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Forbidden","message":"mTLS authentication required"}`, http.StatusForbidden)
			return
		}

		// Validate that required fields are present
		if authCtx.MemberID == "" || authCtx.CertSubject == "" {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Forbidden","message":"Missing certificate information"}`, http.StatusForbidden)
			return
		}

		// Add auth context to request context
		ctx := WithAuthContext(r.Context(), authCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
