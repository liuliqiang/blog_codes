package main

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
)

func chanWriter(opChan chan string) {
	for index := 0; index < 5; index++ {
		opChan <- fmt.Sprintf("num: %d", index)
		time.Sleep(time.Second)
	}
	close(opChan)
}

func main() {
	api := gin.Default()
	api.GET("/download", func(c *gin.Context) {
		opChan := make(chan string)
		go chanWriter(opChan)

		c.Stream(func(w io.Writer) bool {
			output, ok := <-opChan
			if !ok {
				return false
			}
			outputBytes := bytes.NewBufferString(output + "\n")
			c.Writer.Write(outputBytes.Bytes())
			return true
		})
	})

	api.Run(":8080")
}
