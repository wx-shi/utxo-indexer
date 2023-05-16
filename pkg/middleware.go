package pkg

import (
	"bytes"
	"io/ioutil"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LogMiddleware 日志中间件
func LogMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		data, _ := c.GetRawData()
		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(data)) // 关键点

		c.Next()
		duration := time.Since(start)
		logger.Info(path,
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.ByteString("body", data),
			zap.Duration("duration", duration))
	}
}

// CORSMiddleware 跨域中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Max")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		c.Next()
	}
}
