## Description

Brief description of the change and why it's needed.

## Type of change

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Refactor / code cleanup

## Checklist

- [ ] I have read and followed the [AGENTS.md](../AGENTS.md) clean-room policy
- [ ] No GitLab EE source was consulted to derive this change
- [ ] `go build ./... && go test ./... && go vet ./...` all pass
- [ ] `gofmt -l .` is empty
- [ ] No secret values (passwords, tokens, keys) are logged or printed
- [ ] Conventional Commit prefix used in the commit message
- [ ] One logical change per PR (no unrelated commits)

## Related issues

Closes #(issue number)