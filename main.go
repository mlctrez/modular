package main

import (
	"flag"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kevinburke/ssh_config"
	"github.com/mitchellh/go-homedir"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	m := &modular{}
	m.Run()
}

type modular struct {
	repo *git.Repository
}

func (m *modular) Run() {
	m.repo = gitRepo(".")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: modular <arg>")
		fmt.Fprintln(os.Stderr, "  where <arg> is a semantic version or `bump` increment the revision")
		os.Exit(-1)
	}
	var refs []plumbing.Hash
	for _, arg := range args {
		switch arg {
		case "bump":
			refs = append(refs, m.bump(0))
		default:
			// parse out tag if present
			if newTag, err := ParseVersionString(arg); err == nil {
				refs = append(refs, m.createTag(newTag))
				continue
			}
			// this is a commit message
			refs = append(refs, m.commit(arg))
		}
	}
	if len(refs) > 0 {
		log.Println("pushing", refs)
		m.push(refs)
	}
}

func gitRepo(dir string) (repo *git.Repository) {
	var err error
	if repo, err = git.PlainOpen("."); err != nil {
		log.Fatal(err)
	}
	return
}

func (m *modular) latestTag() (latest SemVerTag) {
	var iter storer.ReferenceIter
	var err error
	if iter, err = m.repo.Tags(); err != nil {
		log.Fatal(err)
	}
	latest = SemVerTag{}
	if err = iter.ForEach(func(ref *plumbing.Reference) error {
		refName := string(ref.Name())
		if strings.HasPrefix(refName, "refs/tags/v") {
			tagName := strings.TrimPrefix(refName, "refs/tags/v")
			var tag SemVerTag
			if tag, err = ParseVersionString(tagName); err == nil {
				if tag.GreaterThan(latest) {
					latest = tag
				}
			}
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}
	return
}

func (m *modular) bump(which int) plumbing.Hash {
	tag := m.latestTag()
	switch which {
	case 0:
		tag.Revision++
	case 1:
		tag.Minor++
		tag.Revision = 0
	case 2:
		tag.Major++
		tag.Minor = 0
		tag.Revision = 0
	}
	return m.createTag(tag)
}

func (m *modular) createTag(tag SemVerTag) (ref plumbing.Hash) {
	var head *plumbing.Reference
	var err error
	if head, err = m.repo.Head(); err != nil {
		log.Fatal(err)
	}
	opts := &git.CreateTagOptions{Message: tag.String()}
	var plum *plumbing.Reference
	if plum, err = m.repo.CreateTag(tag.String(), head.Hash(), opts); err != nil {
		log.Fatal(err)
	}
	ref = plum.Hash()
	return
}

func (m *modular) commit(msg string) (ref plumbing.Hash) {
	var worktree *git.Worktree
	var err error
	if worktree, err = m.repo.Worktree(); err != nil {
		log.Fatal(err)
	}
	if ref, err = worktree.Commit(msg, &git.CommitOptions{}); err != nil {
		log.Fatal(err)
	}
	return
}

func (m *modular) push(refs []plumbing.Hash) {
	identity := ssh_config.Get("github.com", "IdentityFile")
	if identity == "" {
		identity = "~/.git/id_rsa"
	}
	expand, err := homedir.Expand(identity)
	if err != nil {
		log.Fatal(err)
	}
	callback, err := ssh.NewKnownHostsCallback()
	if err != nil {
		log.Fatal(err)
	}
	auth, err := ssh.NewPublicKeysFromFile("git", expand, "")
	if err != nil {
		log.Fatal(err)
	}
	clientConfig, err := auth.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}
	clientConfig.HostKeyCallback = callback
	err = m.repo.Push(&git.PushOptions{
		Auth:       auth,
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec("refs/tags/*:refs/tags/*")},
	})
	if err != nil {
		log.Fatal(err)
	}

}

type SemVerTag struct {
	Major    int
	Minor    int
	Revision int
}

func ParseVersionString(in string) (tag SemVerTag, err error) {
	parts := strings.Split(in, ".")
	if len(parts) != 3 {
		err = fmt.Errorf("not in three parts")
		return
	}
	tag.Major, err = strconv.Atoi(parts[0])
	if err != nil {
		tag.Major = 0
		err = fmt.Errorf("major not int")
	}
	tag.Minor, err = strconv.Atoi(parts[1])
	if err != nil {
		tag.Minor = 0
		err = fmt.Errorf("minor not int")
	}
	tag.Revision, err = strconv.Atoi(parts[2])
	if err != nil {
		tag.Revision = 0
		err = fmt.Errorf("revision not int")
	}
	return
}

func (sv SemVerTag) String() string {
	return fmt.Sprintf("v%d.%d.%d", sv.Major, sv.Minor, sv.Revision)
}

func (sv SemVerTag) GreaterThan(other SemVerTag) bool {
	if sv.Major > other.Major {
		return true
	}
	if sv.Minor > other.Minor {
		return true
	}
	if sv.Revision > other.Revision {
		return true
	}
	return false
}
