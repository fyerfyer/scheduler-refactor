package api

import (
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/fyerfyer/scheduler-refactor/common"
)

// success 返回成功响应
func success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, common.ApiResponse{
		Code:    common.ApiSuccess,
		Message: "success",
		Data:    data,
	})
}

// failure 返回失败响应
func failure(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, common.ApiResponse{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}
