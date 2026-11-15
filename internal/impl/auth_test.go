package impl

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/stretchr/testify/require"
	xssh "golang.org/x/crypto/ssh"
)

func writeKey(t *testing.T, home, passphrase string) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	var block *pem.Block
	if passphrase == "" {
		block, err = xssh.MarshalPrivateKey(priv, "")
	} else {
		block, err = xssh.MarshalPrivateKeyWithPassphrase(priv, "", []byte(passphrase))
	}
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".ssh"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".ssh", "id_ed25519"), pem.EncodeToMemory(block), 0o600))
}

func TestAuthenticate(t *testing.T) {
	req := require.New(t)
	const gh = "https://github.com/o/r.git"

	home := t.TempDir()
	_, err := Authenticate(gh, home)
	req.ErrorContains(err, "~/.github_token")

	req.NoError(os.WriteFile(filepath.Join(home, ".github_token"), []byte(" \n"), 0o600))
	_, err = Authenticate(gh, home)
	req.ErrorContains(err, "~/.github_token")

	req.NoError(os.WriteFile(filepath.Join(home, ".github_token"), []byte("tok\n"), 0o600))
	auth, err := Authenticate(gh, home)
	req.NoError(err)
	req.Equal(&http.BasicAuth{Username: "tok"}, auth)

	// any other url uses the ssh key
	_, err = Authenticate("git@gitea.local:o/r.git", home)
	req.ErrorContains(err, "id_ed25519")

	writeKey(t, home, "")
	auth, err = Authenticate("git@gitea.local:o/r.git", home)
	req.NoError(err)
	req.IsType(&ssh.PublicKeys{}, auth)
	req.Equal("git", auth.(*ssh.PublicKeys).User)

	writeKey(t, home, "secret")
	_, err = Authenticate("ssh://git@gitea.local:2222/o/r.git", home)
	req.ErrorContains(err, "id_ed25519")
}

func TestDefaultAuth(t *testing.T) {
	req := require.New(t)
	home := t.TempDir()
	writeKey(t, home, "")
	t.Setenv("HOME", home)
	auth, err := DefaultAuth("git@host:o/r.git")
	req.NoError(err)
	req.IsType(&ssh.PublicKeys{}, auth)

	t.Setenv("HOME", "")
	_, err = DefaultAuth("git@host:o/r.git")
	req.Error(err)
}

func TestOriginURL(t *testing.T) {
	req := require.New(t)
	repo := newRepo(t, "git@host:o/r.git")
	url, err := OriginURL(repo)
	req.NoError(err)
	req.Equal("git@host:o/r.git", url)

	cfg, _ := repo.Config()
	cfg.Remotes["origin"].URLs = []string{"a", "b"}
	req.NoError(repo.SetConfig(cfg))
	_, err = OriginURL(repo)
	req.ErrorContains(err, "single origin")

	delete(cfg.Remotes, "origin")
	req.NoError(repo.SetConfig(cfg))
	_, err = OriginURL(repo)
	req.ErrorContains(err, "no remote named origin")

	cfg.Remotes["other"] = &config.RemoteConfig{Name: "other", URLs: []string{"x"}}
	req.NoError(repo.SetConfig(cfg))
	_, err = OriginURL(repo)
	req.ErrorContains(err, "no remote named origin")
}
