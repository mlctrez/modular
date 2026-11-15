# Committing with modular

Commit, tag and push with `modular`, never with separate `git commit`, `git tag` or `git push`. Docs: https://github.com/mlctrez/modular

```bash
git add <files>
modular [-q] bump|<version> "<commit message>"
```

- `bump` increments the last number of the highest `vX.Y.Z` tag on the repo or origin (`v2.0.9` becomes `v2.0.10`).
- `<version>` such as `1.0.0` creates tag `v1.0.0`. Do not type the `v`.
- `-q` hides success lines. Put it first.

## Rules

- Stage only files that belong in the commit.
- Default to `bump` unless the user names a version.
- It pushes to `origin`. Run it only when the user wants the change pushed and tagged.
- Checks run before anything changes. On `preflight FAILED`, nothing was committed. Fix the cause and run again.
- If the output says `local commit ... not pushed`, the commit and tag already exist. Do not run modular again. Report the error to the user.

## Commit message

First line: imperative summary, about 50 characters. Blank line. Then one bullet per change. Pass it as one quoted argument with real newlines.

```bash
modular bump "back up installed service before deploy

- copy installed binary to <name>.<YYYYMMDD_HHMMSS>.bak before deploy
- keep only the 5 newest backups
- add tests for backup and pruning"
```
