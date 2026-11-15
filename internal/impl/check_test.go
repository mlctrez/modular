package impl

import (
	"os/exec"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"
)

func TestCheckLocal(t *testing.T) {
	req := require.New(t)
	repo := newRepo(t, "unused")

	_, err := CheckLocal(repo)
	req.ErrorContains(err, "no commits")

	stage(t, repo, "a", "1")
	commitStaged(t, repo, "init")
	_, err = CheckLocal(repo)
	req.ErrorContains(err, "nothing staged")

	stage(t, repo, "a", "2")
	head, err := CheckLocal(repo)
	req.NoError(err)
	req.True(head.Name().IsBranch())

	req.NoError(repo.Storer.SetReference(plumbing.NewHashReference(plumbing.HEAD, head.Hash())))
	_, err = CheckLocal(repo)
	req.ErrorContains(err, "detached")
}

func TestCheckLocalNeedsIdentity(t *testing.T) {
	// keep any global git config on the machine from supplying an identity
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	req := require.New(t)
	repo, err := git.PlainInit(t.TempDir(), false)
	req.NoError(err)
	stage(t, repo, "a", "1")
	wt, _ := repo.Worktree()
	_, err = wt.Commit("init", &git.CommitOptions{Author: &object.Signature{Name: "n", Email: "e"}})
	req.NoError(err)
	stage(t, repo, "a", "2")

	_, err = CheckLocal(repo)
	req.ErrorContains(err, "user.name")

	cfg, _ := repo.Config()
	cfg.User.Name = "n"
	req.NoError(repo.SetConfig(cfg))
	_, err = CheckLocal(repo)
	req.ErrorContains(err, "user.email")

	cfg.User.Email = "e"
	req.NoError(repo.SetConfig(cfg))
	_, err = CheckLocal(repo)
	req.NoError(err)
}

func TestCheckLocalMergeInProgress(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	sh := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com"}, args...)...)
		cmd.Dir = dir
		_ = cmd.Run() // the merge is expected to fail
	}
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)
	setUser(t, repo)
	stage(t, repo, "a", "base")
	commitStaged(t, repo, "base")
	sh("checkout", "-qb", "other")
	stage(t, repo, "a", "other")
	commitStaged(t, repo, "other")
	sh("checkout", "-q", "master")
	stage(t, repo, "a", "mine")
	commitStaged(t, repo, "mine")
	sh("merge", "other")

	_, err = CheckLocal(repo)
	require.ErrorContains(t, err, "is in progress (MERGE_HEAD)")
}

func TestCheckRemote(t *testing.T) {
	req := require.New(t)

	// empty origin is fine and has no tags
	empty := newRepo(t, newOrigin(t))
	stage(t, empty, "a", "1")
	commitStaged(t, empty, "init")
	head, _ := empty.Head()
	tags, err := CheckRemote(empty, nil, head)
	req.NoError(err)
	req.Empty(tags)

	// tags on origin are returned without the peeled suffix
	origin := newOrigin(t)
	a := seeded(t, origin)
	pushTag(t, a, "v1.0.0")
	pushTag(t, a, "other")
	head, _ = a.Head()
	tags, err = CheckRemote(a, nil, head)
	req.NoError(err)
	req.ElementsMatch([]string{"v1.0.0", "other"}, tags)

	// ahead of origin fast-forwards
	stage(t, a, "a", "2")
	commitStaged(t, a, "ahead")
	head, _ = a.Head()
	_, err = CheckRemote(a, nil, head)
	req.NoError(err)

	// behind origin: b has a commit a does not have
	b := cloneOf(t, origin)
	stage(t, b, "b", "1")
	commitStaged(t, b, "theirs")
	bHead, _ := b.Head()
	push(t, b, bHead.Name().String()+":"+bHead.Name().String())
	_, err = CheckRemote(a, nil, head)
	req.ErrorContains(err, "pull or rebase")

	// a fetches theirs so the commit is known locally but a has diverged
	req.NoError(a.Fetch(&git.FetchOptions{RemoteName: "origin"}))
	_, err = CheckRemote(a, nil, head)
	req.ErrorContains(err, "pull or rebase")

	// unreachable origin
	cfg, _ := a.Config()
	cfg.Remotes["origin"].URLs = []string{"/nonexistent/repo.git"}
	req.NoError(a.SetConfig(cfg))
	_, err = CheckRemote(a, nil, head)
	req.ErrorContains(err, "cannot reach origin")

	// no origin remote
	delete(cfg.Remotes, "origin")
	req.NoError(a.SetConfig(cfg))
	_, err = CheckRemote(a, nil, head)
	req.Error(err)
}
