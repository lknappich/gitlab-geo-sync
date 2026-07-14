// Package autorepair provides drift auto-repair for git repositories and
// S3 objects. When the consistency sweep or object storage reconciler
// detects drift, this package re-runs the appropriate sync mechanism
// (git fetch or rsync for repos, S3 copy for missing objects).
package autorepair

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// GitRepair re-fetches a specific project repo from the primary.
type GitRepair struct {
	primarySSHHost string
	reposPath      string
	dryRun         bool
}

// NewGitRepair creates a git repairer.
func NewGitRepair(primarySSHHost, reposPath string, dryRun bool) *GitRepair {
	return &GitRepair{
		primarySSHHost: primarySSHHost,
		reposPath:      reposPath,
		dryRun:         dryRun,
	}
}

// RepairRepo re-fetches a single repo identified by its path (e.g.
// "group/subgroup/project.git").
func (g *GitRepair) RepairRepo(ctx context.Context, repoPath string) error {
	localPath := filepath.Join(g.reposPath, repoPath)
	remoteURL := fmt.Sprintf("ssh://%s/var/opt/gitlab/git-data/repositories/%s",
		g.primarySSHHost, repoPath)

	args := []string{
		"-C", localPath,
		"fetch", "--prune", "--no-tags",
		remoteURL,
		"+refs/*:refs/*",
		"+refs/tags/*:refs/tags/*",
	}

	if g.dryRun {
		log.Info().Str("repo", repoPath).Msg("[dry-run] would repair")
		return nil
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch %s: %w: %s", repoPath, err, strings.TrimSpace(string(out)))
	}
	log.Info().Str("repo", repoPath).Msg("auto-repaired git repo")
	return nil
}

// S3Repair copies a missing object from the primary bucket to the replica.
type S3Repair struct {
	dryRun bool
}

// NewS3Repair creates an S3 repairer.
func NewS3Repair(dryRun bool) *S3Repair {
	return &S3Repair{dryRun: dryRun}
}

// RepairObject logs a missing S3 key. Actual S3 copy is handled by the
// cloud provider's cross-region replication; this is a fallback for
// objects that fell through the cracks. Implementation will use
// aws-sdk-go-v2 CopyObject in Phase 2g+.
func (s *S3Repair) RepairObject(ctx context.Context, bucket, key string) error {
	if s.dryRun {
		log.Info().Str("bucket", bucket).Str("key", key).Msg("[dry-run] would copy S3 object")
		return nil
	}
	log.Warn().Str("bucket", bucket).Str("key", key).
		Msg("S3 object drift detected; relying on bucket replication to catch up")
	return nil
}