package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	server := gin.New()

	server.GET("/index", func(c *gin.Context) {

	})

	//apiRouter := server.Group("/api/v1")
	//apiRouter.GET("/posts", func(c *gin.Context) {
	//	c.JSON(http.StatusOK, gin.H{"method": "get"})
	//}).POST("/posts", func(c *gin.Context) {
	//	c.JSON(http.StatusOK, gin.H{"method": "post"})
	//})
	server.GET("/user/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.String(http.StatusOK, "Hello %s", name)
	})

	// However, this one will match /user/john/ and also /user/john/send
	// If no other routers match /user/john, it will redirect to /user/john/
	server.GET("/user/:name/*action", func(c *gin.Context) {
		name := c.Param("name")
		action := c.Param("action")
		message := name + " is " + action
		c.String(http.StatusOK, message)
	})

	// For each matched request Context will hold the route definition
	server.POST("/user/:name/*action", func(c *gin.Context) {
		b := c.FullPath() == "/user/:name/*action" // true
		c.String(http.StatusOK, "%t", b)
	})

	// This handler will add a new router for /user/groups.
	// Exact routes are resolved before param routes, regardless of the order they were defined.
	// Routes starting with /user/groups are never interpreted as /user/:name/... routes
	server.GET("/user/groups", func(c *gin.Context) {
		c.String(http.StatusOK, "The available groups are [...]")
	})
	server.Run("127.0.0.1:8081")
}
