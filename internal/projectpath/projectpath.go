// Package projectpath provides validation for GitLab project paths
// (path_with_namespace) used in webhook triggers and git fetch. It
// rejects path traversal, empty paths, and characters that are not
// valid in GitLab project paths.
package projectpath

import (
	"fmt"
	"regexp"
	"strings"
)

// validPathRe matches GitLab path_with_namespace grammar:
// alphanumeric, underscore, hyphen, dot, and forward-slash separators.
// Must start with an alphanumeric character.
var validPathRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.\-/]*$`)

// Validate checks that a project path is safe for use in filesystem
// operations and SSH URLs. It rejects:
//   - empty paths
//   - paths with leading /
//   - paths containing .. segments (traversal)
//   - paths containing backslashes or NUL
//   - paths that don't match the GitLab path grammar
func Validate(path string) error {
	if path == "" {
		return fmt.Errorf("project path is empty")
	}
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("project path must not start with /")
	}
	if strings.Contains(path, "\\") {
		return fmt.Errorf("project path must not contain backslashes")
	}
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("project path must not contain NUL")
	}
	segments := strings.Split(path, "/")
	for _, seg := range segments {
		if seg == "" {
			return fmt.Errorf("project path contains empty segment")
		}
		if seg == "." || seg == ".." {
			return fmt.Errorf("project path contains traversal segment: %q", seg)
		}
	}
	if !validPathRe.MatchString(path) {
		return fmt.Errorf("project path %q contains invalid characters", path)
	}
	return nil
}
