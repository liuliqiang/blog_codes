package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	server := gin.New()

	//server.GET("/api/v1/posts", func(context *gin.Context) {})
	//server.GET("/api/v1/users", func(context *gin.Context) {})
	//server.GET("/api/v1/users/zhangsan/avatar", func(context *gin.Context) {})
	//server.GET("/api/v1/users/user/*path", func(context *gin.Context) {})
	//server.GET("/api/v1/users/:user", func(context *gin.Context) {})
	//server.GET("/:api", func(context *gin.Context) {})
	//server.GET("/*api", func(context *gin.Context) {})

	server.Run("127.0.0.1:8081")
}

/*

node.path=/*api
node.priority=1
node.fullPath=/*api
node.indices=
node.nType=catchAll
node.wildChild=false
len(node.handlers)=1





*/
