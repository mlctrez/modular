package impl

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

// CheckLocal verifies the repository is in a state where commit, tag and push can succeed.
func CheckLocal(repo *git.Repository) (*plumbing.Reference, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("repository has no commits yet")
	}
	if !head.Name().IsBranch() {
		return nil, fmt.Errorf("HEAD is detached, check out a branch first")
	}
	cfg, err := repo.ConfigScoped(config.SystemScope)
	if err != nil {
		return nil, err
	}
	if cfg.User.Name == "" || cfg.User.Email == "" {
		return nil, fmt.Errorf("git user.name and user.email are not set")
	}
	wt, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	status, err := wt.Status()
	if err != nil {
		return nil, err
	}
	// go-git cannot see conflicted index entries and its commits would drop the second parent,
	// so refuse to run while git is in the middle of an operation.
	// ponytail: only looks in <worktree>/.git, so linked worktrees and submodules are not detected.
	for _, name := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "rebase-merge", "rebase-apply"} {
		if _, err := os.Stat(filepath.Join(wt.Filesystem.Root(), ".git", name)); err == nil {
			return nil, fmt.Errorf("a merge, rebase, cherry-pick or revert is in progress (%s), finish or abort it first", name)
		}
	}
	staged := 0
	for _, st := range status {
		if st.Staging != git.Unmodified && st.Staging != git.Untracked {
			staged++
		}
	}
	if staged == 0 {
		return nil, fmt.Errorf("nothing staged, run git add first")
	}
	return head, nil
}

// CheckRemote contacts origin before anything is committed. It proves the credentials and
// network work, returns origin's tag names, and rejects a branch that would not fast-forward.
func CheckRemote(repo *git.Repository, auth transport.AuthMethod, head *plumbing.Reference) (tags []string, err error) {
	remote, err := repo.Remote("origin")
	if err != nil {
		return nil, err
	}
	refs, err := remote.List(&git.ListOptions{Auth: auth})
	if err != nil && !errors.Is(err, transport.ErrEmptyRemoteRepository) {
		return nil, fmt.Errorf("cannot reach origin: %v", err)
	}
	for _, ref := range refs {
		name := ref.Name().String()
		if strings.HasPrefix(name, "refs/tags/") {
			tags = append(tags, strings.TrimSuffix(strings.TrimPrefix(name, "refs/tags/"), "^{}"))
		}
		if ref.Name() != head.Name() || ref.Hash() == head.Hash() {
			continue
		}
		behind := fmt.Errorf("origin/%s has commits you do not have, pull or rebase first", head.Name().Short())
		theirs, cerr := repo.CommitObject(ref.Hash())
		if cerr != nil {
			return nil, behind
		}
		ours, cerr := repo.CommitObject(head.Hash())
		if cerr != nil {
			return nil, cerr
		}
		if ok, aerr := theirs.IsAncestor(ours); aerr != nil || !ok {
			return nil, behind
		}
	}
	return tags, nil
}
