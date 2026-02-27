package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Admin bypasses all role checks
		if GetCurrentUserIsAdmin(c) {
			c.Next()
			return
		}
		userRole := GetCurrentUserRole(c)
		for _, r := range roles {
			if userRole == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code":    40301,
			"message": "权限不足",
			"data":    nil,
		})
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !GetCurrentUserIsAdmin(c) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    40301,
				"message": "权限不足",
				"data":    nil,
			})
			return
		}
		c.Next()
	}
}
