package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	bankNameParam = ""
	bankName      = ""
	apiPath       = "/api/"
	listenPort    = 8082
)

func main() {
	flag.StringVar(&bankNameParam, "name", bankNameParam, "bank name, eg: citi_bank")
	flag.IntVar(&listenPort, "port", listenPort, "business port")
	flag.Parse()
	apiPath = "/api/" + bankNameParam
	for _, str := range strings.Split(bankNameParam, "_") {
		bankName += strings.Title(str)
	}

	app := gin.New()
	addRoute(app)
	log.Printf("bank %s listening at %d", bankName, listenPort)
	if err := app.Run(fmt.Sprintf(":%d", listenPort)); err != nil {
		log.Fatalf("bank %s listen failed: %v", bankName, err)
	}
	log.Printf("bank %s listen stopped", bankName)
}

func addRoute(app *gin.Engine) {
	app.POST(apiPath+"/TransIn", func(c *gin.Context) {
		log.Printf("TransIn to %s", bankName)
		c.JSON(200, "")
		// c.JSON(409, "") // Status 409 for Failure. Won't be retried
	})
	app.POST(apiPath+"/TransInCompensate", func(c *gin.Context) {
		log.Printf("TransInCompensate to %s", bankName)
		c.JSON(200, "")
	})
	app.POST(apiPath+"/TransOut", func(c *gin.Context) {
		log.Printf("TransOut from %s", bankName)
		c.JSON(200, "")
	})
	app.POST(apiPath+"/TransOutCompensate", func(c *gin.Context) {
		log.Printf("TransOutCompensate from %s", bankName)
		c.JSON(200, "")
	})
}
