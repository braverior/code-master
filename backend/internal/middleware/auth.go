package middleware

import (
	"net/http"
	"strings"

	"github.com/codeMaster/backend/internal/model"
	"github.com/codeMaster/backend/pkg/jwt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AuthMiddleware(jwtSecret string, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string

		// 1. Try Authorization header first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 40101, "message": "Token 格式错误", "data": nil})
				return
			}
		}

		// 2. Fallback to query param (for SSE/EventSource which doesn't support custom headers)
		if tokenStr == "" {
			tokenStr = c.Query("token")
		}

		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 40101, "message": "Token 缺失", "data": nil})
			return
		}

		claims, err := jwt.ParseToken(jwtSecret, tokenStr)
		if err != nil {
			if strings.Contains(err.Error(), "expired") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 40102, "message": "Token 已过期，请重新登录", "data": nil})
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 40103, "message": "Token 无效", "data": nil})
			}
			return
		}

		var user model.User
		if err := db.First(&user, claims.UserID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 40103, "message": "用户不存在", "data": nil})
			return
		}
		if user.Status == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 40104, "message": "用户已禁用", "data": nil})
			return
		}

		c.Set("userID", user.ID)
		c.Set("userRole", user.Role)
		c.Set("isAdmin", user.IsAdmin)
		c.Set("user", &user)
		c.Next()
	}
}

func GetCurrentUser(c *gin.Context) *model.User {
	u, exists := c.Get("user")
	if !exists {
		return nil
	}
	return u.(*model.User)
}

func GetCurrentUserID(c *gin.Context) uint {
	id, _ := c.Get("userID")
	return id.(uint)
}

func GetCurrentUserRole(c *gin.Context) string {
	role, _ := c.Get("userRole")
	return role.(string)
}

func GetCurrentUserIsAdmin(c *gin.Context) bool {
	v, exists := c.Get("isAdmin")
	if !exists {
		return false
	}
	return v.(bool)
}
