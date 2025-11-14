package libtools

import "testing"

func TestParseGitHubRepoFromSSHKeyLine(t *testing.T) {
	var sshKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG5kJv7W0rGMFpjusIyyuwncFsgvlKGRDfiHTqC01bbJ git@github.com:groundhogchain/chuckd.git"
	repo, err := ParseGitHubRepoFromSSHKeyLine(sshKey)
	if err != nil {
		t.Fatal(err)
	}
	if repo != "groundhogchain/chuckd" {
		t.Fatalf("got %s", repo)
	}
}
