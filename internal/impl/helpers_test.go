package impl

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/stretchr/testify/require"
)

// noAuth is for file:// remotes, which need no credentials.
func noAuth(string) (transport.AuthMethod, error) { return nil, nil }

func newOrigin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	_, err := git.PlainInit(dir, true)
	require.NoError(t, err)
	return dir
}

func setUser(t *testing.T, repo *git.Repository) {
	t.Helper()
	cfg, err := repo.Config()
	require.NoError(t, err)
	cfg.User.Name, cfg.User.Email = "tester", "tester@example.com"
	require.NoError(t, repo.SetConfig(cfg))
}

// newRepo creates a repository with origin pointing at url and no commits.
func newRepo(t *testing.T, url string) *git.Repository {
	t.Helper()
	repo, err := git.PlainInit(t.TempDir(), false)
	require.NoError(t, err)
	setUser(t, repo)
	_, err = repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{url}})
	require.NoError(t, err)
	return repo
}

func dirOf(t *testing.T, repo *git.Repository) string {
	t.Helper()
	wt, err := repo.Worktree()
	require.NoError(t, err)
	return wt.Filesystem.Root()
}

// stage writes name and adds it to the index.
func stage(t *testing.T, repo *git.Repository, name, content string) {
	t.Helper()
	wt, err := repo.Worktree()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(wt.Filesystem.Root(), name), []byte(content), 0o644))
	_, err = wt.Add(name)
	require.NoError(t, err)
}

func commitStaged(t *testing.T, repo *git.Repository, msg string) {
	t.Helper()
	wt, err := repo.Worktree()
	require.NoError(t, err)
	_, err = wt.Commit(msg, &git.CommitOptions{})
	require.NoError(t, err)
}

func push(t *testing.T, repo *git.Repository, spec string) {
	t.Helper()
	err := repo.Push(&git.PushOptions{RemoteName: "origin", RefSpecs: []config.RefSpec{config.RefSpec(spec)}})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		require.NoError(t, err)
	}
}

// seeded returns a repository whose first commit is already on origin.
func seeded(t *testing.T, origin string) *git.Repository {
	t.Helper()
	repo := newRepo(t, origin)
	stage(t, repo, "a", "1")
	commitStaged(t, repo, "init")
	head, err := repo.Head()
	require.NoError(t, err)
	push(t, repo, head.Name().String()+":"+head.Name().String())
	return repo
}

// cloneOf returns a second working copy of origin.
func cloneOf(t *testing.T, origin string) *git.Repository {
	t.Helper()
	repo, err := git.PlainClone(t.TempDir(), false, &git.CloneOptions{URL: origin})
	require.NoError(t, err)
	setUser(t, repo)
	return repo
}

// pushTag creates an annotated tag at HEAD and pushes it.
func pushTag(t *testing.T, repo *git.Repository, name string) {
	t.Helper()
	head, err := repo.Head()
	require.NoError(t, err)
	_, err = repo.CreateTag(name, head.Hash(), &git.CreateTagOptions{Message: name})
	require.NoError(t, err)
	push(t, repo, "refs/tags/"+name+":refs/tags/"+name)
}

func commitCount(t *testing.T, repo *git.Repository) (n int) {
	t.Helper()
	iter, err := repo.Log(&git.LogOptions{})
	require.NoError(t, err)
	require.NoError(t, iter.ForEach(func(*object.Commit) error { n++; return nil }))
	return
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
}
