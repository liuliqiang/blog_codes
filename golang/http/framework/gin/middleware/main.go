package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liuliqiang/log4go"
)

func MyLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[FORMATTER TEST] %v | %3d | %13v | %15s | %-7s %#v | %s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
			param.ErrorMessage,
		)
	})
}

func NotUseMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		log4go.Info("This handle should not be invoked")
	}
}

func LiqiangIOMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		end := time.Now()
		log4go.Info("Handler total use %d ms.", end.Sub(start).Milliseconds())
	}
}

func main() {
	server := gin.New()
	server.Use(MyLogger())

	group := server.Group("/api/v1")

	group.GET("/", func(c *gin.Context) {
		c.Error(errors.New("err msg"))
		return
	})

	group.Use(NotUseMiddleware())

	addr := "127.0.0.1:8081"
	server.Run(addr)
}
