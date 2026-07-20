package logging

import (
	"testing"

	"github.com/rs/zerolog"
)

func TestParseLevel(t *testing.T) {
	cases := []struct {
		in   string
		want zerolog.Level
	}{
		{"trace", zerolog.TraceLevel},
		{"TRACE", zerolog.TraceLevel},
		{"debug", zerolog.DebugLevel},
		{"info", zerolog.InfoLevel},
		{"", zerolog.InfoLevel},
		{"warn", zerolog.WarnLevel},
		{"warning", zerolog.WarnLevel},
		{"error", zerolog.ErrorLevel},
		{"fatal", zerolog.FatalLevel},
		{"panic", zerolog.PanicLevel},
		{"unknown", zerolog.InfoLevel},
		{"  info  ", zerolog.InfoLevel},
	}
	for _, tc := range cases {
		got := parseLevel(tc.in)
		if got != tc.want {
			t.Errorf("parseLevel(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseFormat(t *testing.T) {
	cases := []struct {
		in   string
		want Format
	}{
		{"json", FormatJSON},
		{"JSON", FormatJSON},
		{"text", FormatText},
		{"console", FormatText},
		{"pretty", FormatText},
		{"", FormatJSON},
		{"unknown", FormatJSON},
		{"  text  ", FormatText},
	}
	for _, tc := range cases {
		got := parseFormat(tc.in)
		if got != tc.want {
			t.Errorf("parseFormat(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestConfigureReturnsLogger(t *testing.T) {
	Configure("info", "json")
	if zerolog.GlobalLevel() != zerolog.InfoLevel {
		t.Errorf("global level = %v, want info", zerolog.GlobalLevel())
	}
}

func TestConfigureDebugLevel(t *testing.T) {
	Configure("debug", "json")
	if zerolog.GlobalLevel() != zerolog.DebugLevel {
		t.Errorf("global level = %v, want debug", zerolog.GlobalLevel())
	}
}

func TestConfigureTextFormat(t *testing.T) {
	Configure("error", "text")
	if zerolog.GlobalLevel() != zerolog.ErrorLevel {
		t.Errorf("global level = %v, want error", zerolog.GlobalLevel())
	}
}
