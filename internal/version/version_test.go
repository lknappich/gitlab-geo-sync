package version

import (
	"strings"
	"testing"
)

func TestCurrentString(t *testing.T) {
	v := Current()
	s := v.String()
	if !strings.Contains(s, "gitlab-geo-sync") {
		t.Errorf("expected version string to contain 'gitlab-geo-sync', got: %s", s)
	}
}
