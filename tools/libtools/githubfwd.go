package libtools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/groundhogchain/chuckd/lib/liblog"
)

func ForwardSshForGit(ctx context.Context) error {

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	home = filepath.Clean(home)

	// clear .ssh
	dotSsh := filepath.Join(home, ".ssh")
	os.MkdirAll(dotSsh, 0700)
	err = RemoveKeyFiles(dotSsh)
	if err != nil {
		return err
	}

	keys, err := ListSSHAgentPublicKeys()
	if err != nil {
		return err
	}

	var gitconfig bytes.Buffer
	var sshconfig bytes.Buffer
	for _, key := range keys {
		repo, err := ParseGitHubRepoFromSSHKeyLine(key)
		if err != nil {
			liblog.Errorf("skipping %q, does not look like a github deploy key", key)
			continue
		}

		hash := SHA256Hex(key)
		liblog.Infof("adding key-%s for %s", hash, repo)

		fmt.Fprintf(&gitconfig, `
[url "git@key-%s.github.com:%s"]
	insteadOf = https://github.com/%s
	insteadOf = git@github.com:%s
	insteadOf = ssh://git@github.com/%s
`, hash, repo, repo, repo, repo)

		fmt.Fprintf(&sshconfig, `
Host key-%s.github.com
    HostName github.com
    IdentityFile %s/.ssh/key-%s
    IdentitiesOnly yes
`, hash, home, hash)

		err = os.WriteFile(
			filepath.Join(home, fmt.Sprintf(".ssh/key-%s", hash)),
			[]byte(key),
			0644,
		)
		if err != nil {
			return err
		}
	}

	err = os.WriteFile(
		filepath.Join(home, ".gitconfig"),
		gitconfig.Bytes(),
		0644,
	)
	if err != nil {
		return err
	}

	err = os.WriteFile(
		filepath.Join(dotSsh, "config"),
		sshconfig.Bytes(),
		0644,
	)
	if err != nil {
		return err
	}

	return nil
}

// RemoveKeyFiles removes all files in dir whose names match "key-*".
func RemoveKeyFiles(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading directory: %w", err)
	}

	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "key-") {
			continue
		}
		if e.IsDir() {
			continue
		}

		path := filepath.Join(dir, name)
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removing %s: %w", path, err)
		}
	}

	return nil
}

func AppendToFile(path, text string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(text); err != nil {
		return err
	}
	return nil
}

func SHA256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ListSSHAgentPublicKeys runs `ssh-add -L` and returns each line as a string.
// If the agent has no identities, it returns an empty slice and a nil error.
func ListSSHAgentPublicKeys() ([]string, error) {
	cmd := exec.Command("ssh-add", "-L")
	out, err := cmd.CombinedOutput()

	if err != nil {
		msg := strings.TrimSpace(string(out))
		// openSSH prints these on stderr when no keys/agent:
		// - "The agent has no identities."
		// - "Error connecting to agent: ..."
		// - "could not open a connection to your authentication agent."
		if strings.Contains(msg, "agent has no identities") {
			return []string{}, nil
		}
		if strings.Contains(msg, "connecting to agent") || strings.Contains(msg, "could not open") {
			return nil, errors.New(msg)
		}
		// some versions write nothing but still fail—surface whatever we have.
		if msg == "" {
			return nil, fmt.Errorf("ssh-add -L failed: %w", err)
		}
		return nil, errors.New(msg)
	}

	lines := bytes.Split(out, []byte{'\n'})
	var keys []string
	for _, l := range lines {
		s := strings.TrimSpace(string(l))
		if s != "" {
			keys = append(keys, s)
		}
	}
	return keys, nil
}

// ParseGitHubRepoFromSSHKeyLine extracts "org/repo" from lines like:
// ssh-ed25519 AAAA... git@github.com:org/repo.git
func ParseGitHubRepoFromSSHKeyLine(line string) (string, error) {
	parts := strings.Split(strings.TrimSpace(line), " ")
	if len(parts) != 3 {
		return "", fmt.Errorf("ssh key has more than three parts")
	}
	last := parts[2]
	noSuffix, ok := strings.CutSuffix(last, ".git")
	if !ok {
		return "", fmt.Errorf("ssh key comment does not end in .git")
	}
	noPrefix, ok := strings.CutPrefix(noSuffix, "git@github.com:")
	if !ok {
		return "", fmt.Errorf("ssh key comment does not begin with git@github.com:")
	}
	return noPrefix, nil
}
