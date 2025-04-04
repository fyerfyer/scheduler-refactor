package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// 运行测试的时候取消这两个注释

func (s *Server) GetEngine() *gin.Engine {
	return s.engine
}

func (s *Server) GetHTTPEngine() http.Handler {
	return s.engine
}
