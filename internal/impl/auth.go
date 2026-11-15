package impl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// OriginURL returns the single URL configured for the origin remote.
func OriginURL(repo *git.Repository) (string, error) {
	cfg, err := repo.Config()
	if err != nil {
		return "", fmt.Errorf("repository config: %w", err)
	}
	origin, ok := cfg.Remotes["origin"]
	if !ok {
		return "", fmt.Errorf("repository has no remote named origin")
	}
	if len(origin.URLs) != 1 {
		return "", fmt.Errorf("cannot determine single origin url")
	}
	return origin.URLs[0], nil
}

// DefaultAuth picks credentials for originURL from the user's home directory.
func DefaultAuth(originURL string) (transport.AuthMethod, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return Authenticate(originURL, home)
}

// Authenticate uses ~/.github_token for https://github.com remotes and ~/.ssh/id_ed25519 for everything else.
func Authenticate(originURL, home string) (transport.AuthMethod, error) {
	if strings.HasPrefix(originURL, "https://github.com") {
		token, err := os.ReadFile(filepath.Join(home, ".github_token"))
		if err != nil || strings.TrimSpace(string(token)) == "" {
			return nil, fmt.Errorf("unable to get token from ~/.github_token")
		}
		return &http.BasicAuth{Username: strings.TrimSpace(string(token))}, nil
	}
	keys, err := ssh.NewPublicKeysFromFile("git", filepath.Join(home, ".ssh", "id_ed25519"), "")
	if err != nil {
		return nil, fmt.Errorf("ssh key ~/.ssh/id_ed25519: %w", err)
	}
	return keys, nil
}
