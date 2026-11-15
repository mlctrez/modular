package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	xssh "golang.org/x/crypto/ssh"
)

func exec2(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var o, e bytes.Buffer
	code = run(args, &o, &e)
	return code, o.String(), e.String()
}

func TestRunUsage(t *testing.T) {
	req := require.New(t)

	code, out, errs := exec2(t)
	req.Equal(1, code)
	req.Empty(out)
	req.Contains(errs, "Usage: modular")

	code, _, errs = exec2(t, "-h")
	req.Equal(0, code)
	req.Contains(errs, "Usage: modular")

	code, _, errs = exec2(t, "-nope")
	req.Equal(2, code)
	req.Contains(errs, "Usage: modular")

	code, _, errs = exec2(t, "bump")
	req.Equal(1, code)
	req.Contains(errs, "a commit message is required")
	req.Contains(errs, "run modular -h for usage")
}

func TestRunNotARepository(t *testing.T) {
	t.Chdir(t.TempDir())
	code, out, errs := exec2(t, "bump", "msg")
	require.Equal(t, 1, code)
	require.Empty(t, out)
	require.Contains(t, errs, "repository does not exist")
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

// TestRunEndToEnd drives the real command against a local bare origin, with and without -q.
func TestRunEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	req := require.New(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, err := xssh.MarshalPrivateKey(priv, "")
	req.NoError(err)
	req.NoError(os.MkdirAll(filepath.Join(home, ".ssh"), 0o700))
	req.NoError(os.WriteFile(filepath.Join(home, ".ssh", "id_ed25519"), pem.EncodeToMemory(block), 0o600))

	origin, work := t.TempDir(), t.TempDir()
	git(t, origin, "init", "-q", "--bare")
	git(t, work, "init", "-q")
	git(t, work, "config", "user.name", "t")
	git(t, work, "config", "user.email", "t@example.com")
	git(t, work, "remote", "add", "origin", origin)
	req.NoError(os.WriteFile(filepath.Join(work, "a"), []byte("1"), 0o644))
	git(t, work, "add", "a")
	git(t, work, "commit", "-qm", "init")
	git(t, work, "push", "-q", "origin", "HEAD")
	t.Chdir(work)

	req.NoError(os.WriteFile(filepath.Join(work, "a"), []byte("2"), 0o644))
	git(t, work, "add", "a")
	code, out, errs := exec2(t, "bump", "first")
	req.Equal(0, code, errs)
	req.Contains(out, "tag     v0.0.1\n")
	req.Contains(out, "push    origin v0.0.1  ok\n")

	req.NoError(os.WriteFile(filepath.Join(work, "a"), []byte("3"), 0o644))
	git(t, work, "add", "a")
	code, out, errs = exec2(t, "-q", "bump", "second")
	req.Equal(0, code, errs)
	req.Empty(out)

	// -q still prints errors
	code, out, errs = exec2(t, "-q", "bump", "nothing staged")
	req.Equal(1, code)
	req.Empty(out)
	req.Contains(errs, "nothing staged")
}
