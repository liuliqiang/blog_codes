package main

import (
	"flag"
	"log"
	"time"
)

// github.com/liuliqiang/blog-demos/pracitce/go-args/simple-01.go
// made by https://liqiang.io
func main() {
	var sleepDuration = time.Second
	flag.DurationVar(&sleepDuration, "sleep.duration", sleepDuration, "sleep duration")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Ready to sleep")
	time.Sleep(sleepDuration)
	log.Println("Sleep done!")
}
