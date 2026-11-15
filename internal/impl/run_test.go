package impl

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/stretchr/testify/require"
)

func runIn(repo *git.Repository, out *bytes.Buffer, args ...string) error {
	root := ""
	if wt, err := repo.Worktree(); err == nil {
		root = wt.Filesystem.Root()
	}
	return Run(args, Options{Dir: root, Out: out, Auth: noAuth})
}

func originRefs(t *testing.T, origin string) map[string]bool {
	t.Helper()
	repo, err := git.PlainOpen(origin)
	require.NoError(t, err)
	refs, err := repo.References()
	require.NoError(t, err)
	got := map[string]bool{}
	require.NoError(t, refs.ForEach(func(r *plumbing.Reference) error {
		got[r.Name().String()] = true
		return nil
	}))
	return got
}

func TestRunBumpAndExplicitVersion(t *testing.T) {
	req := require.New(t)
	origin := newOrigin(t)
	repo := seeded(t, origin)
	pushTag(t, repo, "v2.0.9")

	var out bytes.Buffer
	stage(t, repo, "a", "2")
	req.NoError(runIn(repo, &out, "bump", "add thing\n\n- detail"))
	head, _ := repo.Head()
	req.Equal(fmt.Sprintf("commit  %s  add thing\ntag     v2.0.9 -> v2.0.10\npush    origin master  ok\npush    origin v2.0.10  ok\n",
		head.Hash().String()[:7]), out.String())
	refs := originRefs(t, origin)
	req.True(refs["refs/tags/v2.0.10"])
	req.True(refs["refs/heads/master"])

	out.Reset()
	stage(t, repo, "a", "3")
	req.NoError(runIn(repo, &out, "3.0.0", "second"))
	req.Contains(out.String(), "tag     v3.0.0\n")
	req.NotContains(out.String(), "->")
	req.True(originRefs(t, origin)["refs/tags/v3.0.0"])
}

func TestRunWithoutTag(t *testing.T) {
	req := require.New(t)
	origin := newOrigin(t)
	repo := seeded(t, origin)
	var out bytes.Buffer
	stage(t, repo, "a", "2")
	req.NoError(runIn(repo, &out, "just a commit"))
	req.NotContains(out.String(), "tag")
	req.Contains(out.String(), "push    origin master  ok")
}

func TestRunFirstBumpWithNoTags(t *testing.T) {
	req := require.New(t)
	repo := seeded(t, newOrigin(t))
	var out bytes.Buffer
	stage(t, repo, "a", "2")
	req.NoError(runIn(repo, &out, "bump", "m"))
	req.Contains(out.String(), "tag     v0.0.1\n")
}

func TestRunBumpSeesTagsOnlyOnOrigin(t *testing.T) {
	req := require.New(t)
	origin := newOrigin(t)
	a := seeded(t, origin)
	b := cloneOf(t, origin)
	pushTag(t, b, "v5.0.0")

	var out bytes.Buffer
	stage(t, a, "a", "2")
	req.NoError(runIn(a, &out, "bump", "m"))
	req.Contains(out.String(), "tag     v5.0.0 -> v5.0.1\n")
}

func TestRunToEmptyOrigin(t *testing.T) {
	req := require.New(t)
	origin := newOrigin(t)
	repo := newRepo(t, origin)
	stage(t, repo, "a", "1")
	commitStaged(t, repo, "init")
	stage(t, repo, "a", "2")
	var out bytes.Buffer
	req.NoError(runIn(repo, &out, "bump", "first push"))
	req.True(originRefs(t, origin)["refs/tags/v0.0.1"])
}

func TestRunPreflightFailuresLeaveRepoUntouched(t *testing.T) {
	origin := newOrigin(t)
	a := seeded(t, origin)
	pushTag(t, a, "v1.0.0")
	cases := map[string]struct {
		args []string
		want string
		prep func(t *testing.T, r *git.Repository)
	}{
		"nothing staged": {[]string{"bump", "m"}, "nothing staged", nil},
		"tag exists locally": {[]string{"1.0.0", "m"}, "tag v1.0.0 already exists locally", func(t *testing.T, r *git.Repository) {
			stage(t, r, "a", "x")
		}},
		"tag exists on origin": {[]string{"7.0.0", "m"}, "tag v7.0.0 already exists on origin", func(t *testing.T, r *git.Repository) {
			stage(t, r, "a", "x")
			pushTag(t, cloneOf(t, origin), "v7.0.0") // pushed after r was cloned, so only origin has it
		}},
		"no origin url": {[]string{"bump", "m"}, "no remote named origin", func(t *testing.T, r *git.Repository) {
			stage(t, r, "a", "x")
			cfg, _ := r.Config()
			delete(cfg.Remotes, "origin")
			require.NoError(t, r.SetConfig(cfg))
		}},
		"unreachable origin": {[]string{"bump", "m"}, "cannot reach origin", func(t *testing.T, r *git.Repository) {
			stage(t, r, "a", "x")
			cfg, _ := r.Config()
			cfg.Remotes["origin"].URLs = []string{"/nonexistent/x.git"}
			require.NoError(t, r.SetConfig(cfg))
		}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repo := cloneOf(t, origin)
			if tc.prep != nil {
				tc.prep(t, repo)
			}
			before := commitCount(t, repo)
			err := runIn(repo, &bytes.Buffer{}, tc.args...)
			require.ErrorContains(t, err, tc.want)
			require.Equal(t, before, commitCount(t, repo))
		})
	}
}

func TestRunBehindOrigin(t *testing.T) {
	req := require.New(t)
	origin := newOrigin(t)
	a := seeded(t, origin)
	b := cloneOf(t, origin)
	stage(t, b, "b", "1")
	commitStaged(t, b, "theirs")
	head, _ := b.Head()
	push(t, b, head.Name().String()+":"+head.Name().String())

	stage(t, a, "a", "2")
	before := commitCount(t, a)
	err := runIn(a, &bytes.Buffer{}, "bump", "m")
	req.ErrorContains(err, "pull or rebase")
	req.Equal(before, commitCount(t, a))
}

func TestRunErrors(t *testing.T) {
	req := require.New(t)

	var ue *UsageError
	req.ErrorAs(Run([]string{"bump"}, Options{}), &ue)

	err := Run([]string{"m"}, Options{Dir: t.TempDir()})
	req.ErrorIs(err, git.ErrRepositoryNotExists)

	repo := seeded(t, newOrigin(t))
	stage(t, repo, "a", "2")
	err = Run([]string{"m"}, Options{Dir: dirOf(t, repo), Auth: func(string) (transport.AuthMethod, error) {
		return nil, fmt.Errorf("no key")
	}})
	req.ErrorContains(err, "preflight FAILED: no key")
	req.Equal(1, commitCount(t, repo))

	// default auth is used when none is injected; an empty HOME has no ssh key
	t.Setenv("HOME", t.TempDir())
	err = Run([]string{"m"}, Options{Dir: dirOf(t, repo)})
	req.ErrorContains(err, "id_ed25519")
}

func TestRunRejectedPushReportsLocalState(t *testing.T) {
	requireGit(t)
	req := require.New(t)
	origin := newOrigin(t)
	repo := seeded(t, origin)

	hook := filepath.Join(origin, "hooks", "pre-receive")
	req.NoError(os.MkdirAll(filepath.Dir(hook), 0o755))
	req.NoError(os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755))

	stage(t, repo, "a", "2")
	var out bytes.Buffer
	err := runIn(repo, &out, "bump", "rejected")
	req.ErrorContains(err, "push origin master FAILED")
	var pe *partialError
	req.ErrorAs(err, &pe)
	req.NotNil(errors.Unwrap(err))
	head, _ := repo.Head()
	req.ErrorContains(err, fmt.Sprintf("local commit %s and tag v0.0.1 were created; not pushed", head.Hash().String()[:7]))
	req.Equal(2, commitCount(t, repo))
	req.NotContains(out.String(), "push")
}

func TestRunTagPushRejectedReportsTagOnly(t *testing.T) {
	requireGit(t)
	req := require.New(t)
	origin := newOrigin(t)
	repo := seeded(t, origin)

	hook := filepath.Join(origin, "hooks", "update")
	req.NoError(os.MkdirAll(filepath.Dir(hook), 0o755))
	req.NoError(os.WriteFile(hook, []byte("#!/bin/sh\ncase \"$1\" in refs/tags/*) exit 1;; esac\n"), 0o755))

	stage(t, repo, "a", "2")
	var out bytes.Buffer
	err := runIn(repo, &out, "bump", "tag rejected")
	req.ErrorContains(err, "push origin v0.0.1 FAILED")
	req.ErrorContains(err, "local tag v0.0.1 was created; not pushed")
	req.Contains(out.String(), "push    origin master  ok")
}

func TestPushRefUpToDate(t *testing.T) {
	req := require.New(t)
	repo := seeded(t, newOrigin(t))
	head, _ := repo.Head()
	spec := head.Name().String() + ":" + head.Name().String()
	req.NoError(pushRef(repo, nil, "master", config.RefSpec(spec)))
	cfg, _ := repo.Config()
	cfg.Remotes["origin"].URLs = []string{"/nonexistent/x.git"}
	req.NoError(repo.SetConfig(cfg))
	req.ErrorContains(pushRef(repo, nil, "bad", config.RefSpec(spec)), "push origin bad FAILED")
}

func TestHelpersOnBrokenRepo(t *testing.T) {
	req := require.New(t)
	repo := newRepo(t, "unused") // no commits
	req.Error(createTag(repo, SemVerTag{Parsed: true}))
	_, err := commit(repo, "nothing staged")
	req.Error(err)
	names, err := localTagNames(repo)
	req.NoError(err)
	req.Empty(names)
}
