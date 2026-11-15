// Package impl implements modular's commit, tag and push workflow.
package impl

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

// Options configures Run. The zero value works on the current directory with no progress output.
type Options struct {
	Dir  string                                               // repository directory, default "."
	Out  io.Writer                                            // progress lines, default discard
	Auth func(originURL string) (transport.AuthMethod, error) // default DefaultAuth
}

// partialError reports a failure after local changes were made.
type partialError struct {
	err  error
	note string
}

func (e *partialError) Error() string { return e.err.Error() + "\n" + e.note }
func (e *partialError) Unwrap() error { return e.err }

func short(h plumbing.Hash) string { return h.String()[:7] }

// Run commits the staged files, optionally tags, and pushes to origin.
// Every check that can fail runs before the repository is modified.
func Run(args []string, o Options) error {
	req, err := ParseArgs(args)
	if err != nil {
		return err
	}
	if o.Dir == "" {
		o.Dir = "."
	}
	if o.Out == nil {
		o.Out = io.Discard
	}
	if o.Auth == nil {
		o.Auth = DefaultAuth
	}
	repo, err := git.PlainOpen(o.Dir)
	if err != nil {
		return err
	}

	preflight := func(err error) error { return fmt.Errorf("preflight FAILED: %w", err) }
	head, err := CheckLocal(repo)
	if err != nil {
		return preflight(err)
	}
	url, err := OriginURL(repo)
	if err != nil {
		return preflight(err)
	}
	auth, err := o.Auth(url)
	if err != nil {
		return preflight(err)
	}
	remoteTags, err := CheckRemote(repo, auth, head)
	if err != nil {
		return preflight(err)
	}
	localTags, err := localTagNames(repo)
	if err != nil {
		return preflight(err)
	}

	tag, prev := req.Tag, SemVerTag{}
	if req.Bump {
		prev = Highest(append(append([]string{}, localTags...), remoteTags...))
		tag = prev.NextRevision()
	}
	if tag.Parsed {
		if contains(localTags, tag.String()) {
			return preflight(fmt.Errorf("tag %s already exists locally", tag))
		}
		if contains(remoteTags, tag.String()) {
			return preflight(fmt.Errorf("tag %s already exists on origin", tag))
		}
	}

	hash, err := commit(repo, req.Message)
	if err != nil {
		return fmt.Errorf("commit FAILED: %w", err)
	}
	note := fmt.Sprintf("local commit %s was created; not pushed", short(hash))
	partial := func(err error) error { return &partialError{err, note} }
	fmt.Fprintf(o.Out, "commit  %s  %s\n", short(hash), strings.SplitN(req.Message, "\n", 2)[0])

	if tag.Parsed {
		if err = createTag(repo, tag); err != nil {
			return partial(fmt.Errorf("tag FAILED: %w", err))
		}
		note = fmt.Sprintf("local commit %s and tag %s were created; not pushed", short(hash), tag)
		if prev.Parsed {
			fmt.Fprintf(o.Out, "tag     %s -> %s\n", prev, tag)
		} else {
			fmt.Fprintf(o.Out, "tag     %s\n", tag)
		}
	}

	branch := config.RefSpec(fmt.Sprintf("%s:%s", head.Name(), head.Name()))
	if err = pushRef(repo, auth, head.Name().Short(), branch); err != nil {
		return partial(err)
	}
	fmt.Fprintf(o.Out, "push    origin %s  ok\n", head.Name().Short())
	if tag.Parsed {
		note = fmt.Sprintf("local tag %s was created; not pushed", tag)
		if err = pushRef(repo, auth, tag.String(), tag.RefSpec()); err != nil {
			return partial(err)
		}
		fmt.Fprintf(o.Out, "push    origin %s  ok\n", tag)
	}
	return nil
}

func localTagNames(repo *git.Repository) (names []string, err error) {
	iter, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("reading tags: %w", err)
	}
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		names = append(names, ref.Name().Short())
		return nil
	})
	return
}

func commit(repo *git.Repository, msg string) (plumbing.Hash, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return wt.Commit(msg, &git.CommitOptions{})
}

func createTag(repo *git.Repository, tag SemVerTag) error {
	head, err := repo.Head()
	if err != nil {
		return err
	}
	_, err = repo.CreateTag(tag.String(), head.Hash(), &git.CreateTagOptions{Message: tag.String()})
	return err
}

func pushRef(repo *git.Repository, auth transport.AuthMethod, name string, spec config.RefSpec) error {
	err := repo.Push(&git.PushOptions{Auth: auth, RemoteName: "origin", RefSpecs: []config.RefSpec{spec}})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("push origin %s FAILED: %w", name, err)
	}
	return nil
}
