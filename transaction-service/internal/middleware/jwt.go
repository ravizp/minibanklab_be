package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims holds the JWT token claims extracted from a validated token.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// AuthErrorResponse provides a consistent JSON error structure for auth middleware responses.
type AuthErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// JWTAuth returns a Gin middleware that validates Bearer JWT tokens.
// It accepts the Authorization header in two formats for Swagger and inter-service compatibility:
//   - "Bearer <token>" (standard RFC 6750)
//   - "<token>" (raw token, for convenience)
func JWTAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, AuthErrorResponse{
				Error:   "MISSING_TOKEN",
				Message: "Authorization header is required. Use format: Bearer <token>",
			})
			return
		}

		tokenString := extractBearerToken(authHeader)
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, AuthErrorResponse{
				Error:   "EMPTY_TOKEN",
				Message: "Token value is empty after parsing the Authorization header",
			})
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		},
			jwt.WithIssuer("minibank-user-service"),
			jwt.WithExpirationRequired(),
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		)

		if err != nil {
			status, resp := classifyJWTError(err)
			c.AbortWithStatusJSON(status, resp)
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, AuthErrorResponse{
				Error:   "INVALID_CLAIMS",
				Message: "Token claims could not be verified",
			})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Next()
	}
}

// extractBearerToken extracts the JWT token string from the Authorization header value.
// Supports "Bearer <token>" (case-insensitive) and raw "<token>" formats.
func extractBearerToken(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	return authHeader
}

// classifyJWTError maps JWT parsing errors to appropriate HTTP status codes and descriptive messages.
func classifyJWTError(err error) (int, AuthErrorResponse) {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return http.StatusUnauthorized, AuthErrorResponse{
			Error:   "TOKEN_EXPIRED",
			Message: "Token has expired, please login again to obtain a new token",
		}
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return http.StatusUnauthorized, AuthErrorResponse{
			Error:   "TOKEN_NOT_ACTIVE",
			Message: "Token is not yet valid",
		}
	case errors.Is(err, jwt.ErrSignatureInvalid):
		return http.StatusUnauthorized, AuthErrorResponse{
			Error:   "INVALID_SIGNATURE",
			Message: "Token signature verification failed",
		}
	case errors.Is(err, jwt.ErrTokenMalformed):
		return http.StatusUnauthorized, AuthErrorResponse{
			Error:   "MALFORMED_TOKEN",
			Message: "Token format is invalid, ensure you are sending a valid JWT",
		}
	default:
		return http.StatusUnauthorized, AuthErrorResponse{
			Error:   "INVALID_TOKEN",
			Message: "Token validation failed: " + err.Error(),
		}
	}
}
