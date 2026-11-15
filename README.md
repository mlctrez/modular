# modular

### Background

This is a hobby project for learning go-git and simplifying commit - tag - push workflows.

### Usage

modular `tag_command` "commit message"

`tag_command` indicates that an annotated tag should be created. It can be either:

* the keyword `bump` - increment the last number of the highest `vX.Y.Z` tag, so `v2.0.9` becomes `v2.0.10`
* a semver string - the tag will be created with a prefix of `v` - `1.0.0` becomes `v1.0.0`

`main.go` handles flags and exit codes. The workflow lives in `internal/impl` (`args`, `auth`, `check`, `run`, `tags`).

Before changing anything, modular checks that: a message was given, HEAD is on a branch with at least one commit,
`user.name`/`user.email` are set, at least one file is staged and no merge, rebase, cherry-pick or revert is in progress, origin is reachable with
working credentials, the local branch is not behind origin, and the tag does not already exist locally or on origin.
`bump` starts from the highest `vX.Y.Z` tag found locally or on origin. If any check fails, nothing is committed.

Each step prints one line (commit, tag, push). Errors go to stderr with a non-zero exit.
`-q` (before the arguments) suppresses the success lines.

### Agent instructions

`steering/modular.md` tells coding agents to commit with `modular` instead of `git commit`, `git tag` and `git push`.
This repository holds the versioned copy, so edit it here and then copy it to wherever your agent reads instructions:

* Kiro: `cp steering/modular.md ~/.kiro/steering/modular.md` (global), or into `<workspace>/.kiro/steering/` for one project
* Other agents: append or reference the file from the project's `AGENTS.md` (or `CLAUDE.md`, `.cursorrules`, and so on)

### Example

```bash
git add README.md
modular bump "add documentation"
```
replaces
```bash
git add README.md
git tag -l  # find the last tag used and increment the last number
git commit -m "add documentation"
git tag -a v0.1.5 -m "v0.1.5"
git push origin master
git push origin v0.1.5
```
















