package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apperrors "github.com/parking-violation-portal/backend/pkg/errors"
	jwtutil "github.com/parking-violation-portal/backend/pkg/jwt"
)

const userContextKey = "userContext"

type UserContext struct {
	UserID uuid.UUID
	Email  string
	Role   string
}

func Auth(jwtManager *jwtutil.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			abortWithError(c, apperrors.New(apperrors.CodeUnauthorized, "missing authorization header", http.StatusUnauthorized))
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			abortWithError(c, apperrors.New(apperrors.CodeUnauthorized, "invalid authorization header", http.StatusUnauthorized))
			return
		}

		claims, err := jwtManager.Parse(parts[1])
		if err != nil {
			abortWithError(c, apperrors.New(apperrors.CodeUnauthorized, "invalid or expired token", http.StatusUnauthorized))
			return
		}

		userID, err := jwtutil.SubjectUUID(claims)
		if err != nil {
			abortWithError(c, apperrors.New(apperrors.CodeUnauthorized, "invalid token subject", http.StatusUnauthorized))
			return
		}

		c.Set(userContextKey, UserContext{
			UserID: userID,
			Email:  claims.Email,
			Role:   claims.Role,
		})
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		user, ok := UserFromContext(c)
		if !ok {
			abortWithError(c, apperrors.New(apperrors.CodeUnauthorized, "unauthorized", http.StatusUnauthorized))
			return
		}

		if _, ok := allowed[user.Role]; !ok {
			abortWithError(c, apperrors.New(apperrors.CodeForbidden, "forbidden", http.StatusForbidden))
			return
		}

		c.Next()
	}
}

func UserFromContext(c *gin.Context) (UserContext, bool) {
	value, ok := c.Get(userContextKey)
	if !ok {
		return UserContext{}, false
	}
	user, ok := value.(UserContext)
	return user, ok
}

func abortWithError(c *gin.Context, err *apperrors.AppError) {
	c.AbortWithStatusJSON(err.HTTPStatus, gin.H{
		"error": gin.H{
			"code":    err.Code,
			"message": err.Message,
			"details": err.Details,
		},
	})
}
