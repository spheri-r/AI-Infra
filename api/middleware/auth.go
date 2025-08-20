package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"llm-inferra/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		tokenString := tokenParts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		userID, ok := claims["user_id"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			c.Abort()
			return
		}

		username, ok := claims["username"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username in token"})
			c.Abort()
			return
		}

		role, ok := claims["role"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid role in token"})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", uint(userID))
		c.Set("username", username)
		c.Set("role", models.UserRole(role))

		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
			c.Abort()
			return
		}

		userRole, ok := role.(models.UserRole)
		if !ok || userRole != models.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Helper function to get user ID from context
func GetUserID(c *gin.Context) (uint, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, jwt.ErrTokenInvalidClaims
	}

	id, ok := userID.(uint)
	if !ok {
		return 0, jwt.ErrTokenInvalidClaims
	}

	return id, nil
}

// Helper function to get user role from context
func GetUserRole(c *gin.Context) (models.UserRole, error) {
	role, exists := c.Get("role")
	if !exists {
		return "", jwt.ErrTokenInvalidClaims
	}

	userRole, ok := role.(models.UserRole)
	if !ok {
		return "", jwt.ErrTokenInvalidClaims
	}

	return userRole, nil
}

// Helper function to check if user is admin
func IsAdmin(c *gin.Context) bool {
	role, err := GetUserRole(c)
	if err != nil {
		return false
	}
	return role == models.RoleAdmin
}

// Pagination middleware
func PaginationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		page := 1
		limit := 10

		if p := c.Query("page"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
				page = parsed
			}
		}

		if l := c.Query("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}

		offset := (page - 1) * limit

		c.Set("page", page)
		c.Set("limit", limit)
		c.Set("offset", offset)

		c.Next()
	}
}
