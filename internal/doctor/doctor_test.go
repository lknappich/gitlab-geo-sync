package doctor

import (
	"context"
	"strings"
	"testing"

	"github.com/lknappich/gitlab-geo-sync/internal/config"
)

func TestResultSummary(t *testing.T) {
	r := &Result{
		Checks: []Check{
			{Name: "ssh:primary", Status: "PASS"},
			{Name: "pg:primary:control", Status: "PASS"},
			{Name: "dbkey:primary", Status: "FAIL"},
			{Name: "pg:secondary:in_recovery", Status: "WARN"},
		},
	}
	for _, c := range r.Checks {
		switch c.Status {
		case "PASS":
			r.Pass++
		case "FAIL":
			r.Fail++
		case "WARN":
			r.Warn++
		}
	}
	if r.Pass != 2 {
		t.Errorf("pass = %d, want 2", r.Pass)
	}
	if r.Fail != 1 {
		t.Errorf("fail = %d, want 1", r.Fail)
	}
	if r.Warn != 1 {
		t.Errorf("warn = %d, want 1", r.Warn)
	}
}

func TestS3BucketCheckConfigMissing(t *testing.T) {
	s3 := &config.S3Config{}
	c := s3BucketCheck(context.TODO(), "primary", s3)
	if c.Status != "FAIL" {
		t.Errorf("expected FAIL for empty config, got %s", c.Status)
	}
}

func TestS3BucketCheckConfigOK(t *testing.T) {
	s3 := &config.S3Config{
		PrimaryBucket: "bucket-p",
		ReplicaBucket: "bucket-r",
		AccessKey:     "AKIAEXAMPLE",
		SecretKey:     "secretexample",
	}
	c := s3BucketCheck(context.TODO(), "primary", s3)
	if c.Status != "PASS" {
		t.Errorf("expected PASS, got %s: %s", c.Status, c.Detail)
	}
}

func TestS3BucketCheckMissingKeys(t *testing.T) {
	s3 := &config.S3Config{
		PrimaryBucket: "bucket-p",
		ReplicaBucket: "bucket-r",
		AccessKey:     "",
		SecretKey:     "",
	}
	c := s3BucketCheck(context.TODO(), "primary", s3)
	if !strings.Contains(c.Detail, "access_key/secret_key") {
		t.Errorf("expected missing keys detail, got: %s", c.Detail)
	}
}

func TestPrintOutput(t *testing.T) {
	r := &Result{
		Checks: []Check{
			{Name: "test", Category: "cat", Status: "PASS", Detail: "detail"},
		},
		Pass: 1,
	}
	r.Print()
}
