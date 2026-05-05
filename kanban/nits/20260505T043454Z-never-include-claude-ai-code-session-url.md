# never include claude.ai/code/session_* URLs in commit messages or PR bodies

**Captured:** 20260505T043454Z
**Branch:** claude/implement-m10-bootstrap-rWNtD
**Paths:**
- CLAUDE.md

Claude Code's git-commit and gh-pr-create templates embed claude.ai/code/session_<id> URLs by default. Even though the URLs are auth-gated, the permanence of git history plus possible future access-model changes plus metadata leakage make this worth opting out of repo-wide. Fix is a one-line CLAUDE.md rule.
