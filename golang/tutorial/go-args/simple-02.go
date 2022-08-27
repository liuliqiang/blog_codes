package main

import (
	"flag"
	"time"
)

// github.com/liuliqiang/blog-demos/pracitce/go-args/simple-02.go
// made by https://liqiang.io
type TestConfig struct {
	Name     string
	Interval time.Duration
}

func main() {
	var cfg TestConfig
	flag.StringVar(&cfg.Name, "name", "zhangsan", "only name")
	flag.DurationVar(&cfg.Interval, "check.interval", time.Second, "interval to check name")
	flag.Parse()

	CheckName(&cfg)
}
