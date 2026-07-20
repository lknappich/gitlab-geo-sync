package apivalidator

import (
	"testing"

	"github.com/lknappich/gitlab-geo-sync/internal/config"
)

func TestNewBuildsURLs(t *testing.T) {
	cfg := &config.Config{
		Primary: config.SiteConfig{
			ExternalURL: "https://gitlab.primary.example.com/",
		},
		Secondaries: []config.SiteConfig{
			{ExternalURL: "https://gitlab.secondary.example.com/"},
		},
		APIValidator: &config.APIValidatorConfig{
			Enabled:        true,
			PrimaryToken:   "tok1",
			SecondaryToken: "tok2",
		},
	}
	r := New(cfg)
	if r.primaryURL != "https://gitlab.primary.example.com/api/v4" {
		t.Errorf("primaryURL = %q", r.primaryURL)
	}
	if r.secondaryURL != "https://gitlab.secondary.example.com/api/v4" {
		t.Errorf("secondaryURL = %q", r.secondaryURL)
	}
	if r.primaryToken != "tok1" || r.secondaryToken != "tok2" {
		t.Error("tokens not set")
	}
}
