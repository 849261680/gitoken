package tokenheat

import (
	"os/exec"
	"strings"
)

// DetectGitHubUsername tries to determine the user's GitHub username from
// local git/gh configuration. Returns empty string if detection fails.
func DetectGitHubUsername() string {
	if u := detectFromGitConfig(); u != "" {
		return u
	}
	if u := detectFromGhAuth(); u != "" {
		return u
	}
	return ""
}

func detectFromGitConfig() string {
	cmd := exec.Command("git", "config", "--global", "github.user")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	u := strings.TrimSpace(string(out))
	if u == "" {
		return ""
	}
	return u
}

func detectFromGhAuth() string {
	cmd := exec.Command("gh", "auth", "status")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		// Strip leading checkmark/prefix: "✓ Logged in to github.com account USERNAME (keyring)"
		line = strings.TrimLeft(line, "✓✔ *-")
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "Logged in to github.com account "); ok {
			parts := strings.Fields(after)
			if len(parts) > 0 {
				return strings.TrimRight(parts[0], ".")
			}
		}
	}
	return ""
}
