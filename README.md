# modular

[![Go Report](https://badge.mlctrez.com/mlctrez/modular)](https://goreportcard.com/report/github.com/mlctrez/modular)
[![golangci-lint](https://github.com/mlctrez/modular/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/mlctrez/modular/actions/workflows/golangci-lint.yml)

### Background

This is a hobby project for learning go-git and simplifying commit - tag - push workflows.

### Usage

modular `tag_command` "commit message"

`tag_command` indicates that an annotated tag should be created. It can be either:

* the keyword `bump` - increment the minor revision of the latest semver tag in the repo
* a semver string - the tag will be created with a prefix of `v` - `1.0.0` becomes `v1.0.0`

### Example

```bash
git add README.md
modular bump "add documentation"
```
replaces
```bash
git add README.md
git tag -l  # find the last tag used and increment the minor revision
git commit -m "add documentation"
git tag -a v0.1.5 -m "v0.1.5"
git push origin master
git push origin v0.1.5
```
















