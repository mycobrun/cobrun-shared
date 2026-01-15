// Package auth provides authentication middleware.
package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/mycobrun/cobrun-shared/errors"
)

// ContextKey is used for storing auth data in context.
type ContextKey string

const (
	// ClaimsContextKey is the context key for JWT claims.
	ClaimsContextKey ContextKey = "claims"
	// UserIDContextKey is the context key for user ID.
	UserIDContextKey ContextKey = "user_id"
	// TokenContextKey is the context key for the raw JWT token.
	TokenContextKey ContextKey = "token"
	// ServiceTokenContextKey is the context key for service token authentication.
	ServiceTokenContextKey ContextKey = "service_token"
)

// Middleware creates an authentication middleware.
func Middleware(jwtManager *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				errors.WriteError(w, errors.Unauthorized(""), "")
				return
			}

			// Expect "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				errors.WriteError(w, errors.Unauthorized("invalid authorization header format"), "")
				return
			}

			tokenString := parts[1]

			// Validate token
			claims, err := jwtManager.ValidateToken(tokenString)
			if err != nil {
				if err == ErrTokenExpired {
					errors.WriteError(w, errors.Unauthorized("token expired"), "")
				} else {
					errors.WriteError(w, errors.Unauthorized("invalid token"), "")
				}
				return
			}

			// Add claims and token to context
			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			ctx = context.WithValue(ctx, UserIDContextKey, claims.UserID)
			ctx = context.WithValue(ctx, TokenContextKey, tokenString)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// MiddlewareWithServiceToken creates an authentication middleware that supports both
// JWT Bearer tokens and service tokens for internal service-to-service calls.
// The serviceToken parameter is the expected service token value.
func MiddlewareWithServiceToken(jwtManager *JWTManager, serviceToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// First, check for service token (used for internal service-to-service calls)
			if serviceToken != "" {
				if svcToken := r.Header.Get("X-Service-Token"); svcToken != "" {
					if svcToken == serviceToken {
						// Valid service token - mark as service call and proceed
						ctx := context.WithValue(r.Context(), ServiceTokenContextKey, true)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
					// Invalid service token
					errors.WriteError(w, errors.Unauthorized("invalid service token"), "")
					return
				}
			}

			// Fall back to JWT authentication
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				errors.WriteError(w, errors.Unauthorized(""), "")
				return
			}

			// Expect "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				errors.WriteError(w, errors.Unauthorized("invalid authorization header format"), "")
				return
			}

			tokenString := parts[1]

			// Validate token
			claims, err := jwtManager.ValidateToken(tokenString)
			if err != nil {
				if err == ErrTokenExpired {
					errors.WriteError(w, errors.Unauthorized("token expired"), "")
				} else {
					errors.WriteError(w, errors.Unauthorized("invalid token"), "")
				}
				return
			}

			// Add claims and token to context
			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			ctx = context.WithValue(ctx, UserIDContextKey, claims.UserID)
			ctx = context.WithValue(ctx, TokenContextKey, tokenString)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// IsServiceCall checks if the request was authenticated via service token.
func IsServiceCall(ctx context.Context) bool {
	val, ok := ctx.Value(ServiceTokenContextKey).(bool)
	return ok && val
}

// OptionalMiddleware creates an optional authentication middleware.
// It adds claims to context if present but doesn't fail if missing.
func OptionalMiddleware(jwtManager *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					if claims, err := jwtManager.ValidateToken(parts[1]); err == nil {
						ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
						ctx = context.WithValue(ctx, UserIDContextKey, claims.UserID)
						r = r.WithContext(ctx)
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole creates a middleware that checks for specific roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaimsFromContext(r.Context())
			if claims == nil {
				errors.WriteError(w, errors.Unauthorized(""), "")
				return
			}

			hasRole := false
			for _, required := range roles {
				for _, userRole := range claims.Roles {
					if userRole == required {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				errors.WriteError(w, errors.Forbidden("insufficient permissions"), "")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireUserType creates a middleware that checks for specific user types.
func RequireUserType(userTypes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaimsFromContext(r.Context())
			if claims == nil {
				errors.WriteError(w, errors.Unauthorized(""), "")
				return
			}

			allowed := false
			for _, ut := range userTypes {
				if claims.UserType == ut {
					allowed = true
					break
				}
			}

			if !allowed {
				errors.WriteError(w, errors.Forbidden("user type not allowed"), "")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetClaimsFromContext retrieves claims from context.
func GetClaimsFromContext(ctx context.Context) *Claims {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}

// GetClaims is an alias for GetClaimsFromContext for convenience.
func GetClaims(ctx context.Context) *Claims {
	return GetClaimsFromContext(ctx)
}

// GetUserIDFromContext retrieves user ID from context.
func GetUserIDFromContext(ctx context.Context) string {
	userID, ok := ctx.Value(UserIDContextKey).(string)
	if !ok {
		return ""
	}
	return userID
}

// GetTokenFromContext retrieves the raw JWT token from context.
func GetTokenFromContext(ctx context.Context) string {
	token, ok := ctx.Value(TokenContextKey).(string)
	if !ok {
		return ""
	}
	return token
}

// WithClaims adds claims to the context. Useful for testing.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ClaimsContextKey, claims)
}
