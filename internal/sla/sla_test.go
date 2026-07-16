package sla

import (
	"bytes"
	"context"
	"testing"
)

func TestGeneratePrintsReport(t *testing.T) {
	var buf bytes.Buffer
	err := Generate(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	out := buf.String()
	if out == "" {
		t.Error("expected non-empty report")
	}
	if !bytes.Contains(buf.Bytes(), []byte("SLA Report")) {
		t.Errorf("expected 'SLA Report' in output, got: %s", out)
	}
}

func TestReportPrint(t *testing.T) {
	r := &Report{
		ComponentsHealthy: 3,
		ComponentsTotal:   4,
	}
	var buf bytes.Buffer
	r.Print(&buf)
	out := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("3/4")) {
		t.Errorf("expected '3/4' in output, got: %s", out)
	}
}
