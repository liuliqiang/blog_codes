package main

import (
	"fmt"
)

type MakrdownBuilder interface {
	WriteTitle()
	WriteContent()
	GetResult() string
}

type MarkdownDirector struct {
	builder MakrdownBuilder
}

func (d *MarkdownDirector) Build() string {
	d.builder.WriteTitle()
	d.builder.WriteContent()
	return d.builder.GetResult()
}

func NewMarkdownDirector(builder MakrdownBuilder) *MarkdownDirector {
	return &MarkdownDirector{builder: builder}
}

func main() {
	builder := NewPdfMarkdownBuilder()
	director := NewMarkdownDirector(builder)
	fmt.Printf("Result file is: %s", director.Build())
}
