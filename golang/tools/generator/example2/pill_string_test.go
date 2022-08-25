package painkiller_test

import (
	"testing"

	painkiller "github.com/liuliqiang/blog_codes/golang/tools/generator/example2"
)

func TestPill_String(t *testing.T) {
	var p painkiller.Pill

	if p.String() != "Placebo" {
		t.Fatalf("p should equal to Placebo")
	}

	p = painkiller.Aspirin
	if p.String() != "Aspirin" {
		t.Fail()
	}

	p = painkiller.Ibuprofen
	if p.String() != "Ibuprofen" {
		t.Fail()
	}

	p = painkiller.Paracetamol
	if p.String() != "Paracetamol" {
		t.Fail()
	}

	p = painkiller.Acetaminophen
	if p.String() != "Paracetamol" {
		t.Fail()
	}
}
