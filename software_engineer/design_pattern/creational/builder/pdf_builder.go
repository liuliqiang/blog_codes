package main

func NewPdfMarkdownBuilder() MakrdownBuilder {
	return &pdfMarkdownBuilder{}
}

type pdfMarkdownBuilder struct {
}

func (p *pdfMarkdownBuilder) WriteTitle() {
	panic("implement me")
}

func (p *pdfMarkdownBuilder) WriteContent() {
	panic("implement me")
}

func (p *pdfMarkdownBuilder) GetResult() string {
	panic("implement me")
}
