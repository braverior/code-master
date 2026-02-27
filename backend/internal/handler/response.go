package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Response helpers

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    data,
	})
}

func SuccessPaged(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func Error(c *gin.Context, httpCode int, code int, message string) {
	c.JSON(httpCode, gin.H{
		"code":    code,
		"message": message,
		"data":    nil,
	})
}

func BadRequest(c *gin.Context, code int, message string) {
	Error(c, http.StatusBadRequest, code, message)
}

func Unauthorized(c *gin.Context, code int, message string) {
	Error(c, http.StatusUnauthorized, code, message)
}

func Forbidden(c *gin.Context, code int, message string) {
	Error(c, http.StatusForbidden, code, message)
}

func NotFound(c *gin.Context, code int, message string) {
	Error(c, http.StatusNotFound, code, message)
}

func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, 50001, message)
}

func parseID(s string) uint {
	id, _ := strconv.ParseUint(s, 10, 64)
	return uint(id)
}

func parsePage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func parseErrorCode(err error) (int, string) {
	msg := err.Error()
	if len(msg) > 5 && msg[5] == ':' {
		code, e := strconv.Atoi(msg[:5])
		if e == nil {
			return code, msg[6:]
		}
	}
	return 50001, msg
}
