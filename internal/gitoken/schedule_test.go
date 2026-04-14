package gitoken

import (
	"strings"
	"testing"
)

func TestParseScheduleTime(t *testing.T) {
	hour, minute, err := parseScheduleTime("00:05")
	if err != nil {
		t.Fatalf("parseScheduleTime returned error: %v", err)
	}
	if hour != 0 || minute != 5 {
		t.Fatalf("unexpected time: got %02d:%02d", hour, minute)
	}
}

func TestParseScheduleTimeRejectsInvalid(t *testing.T) {
	for _, value := range []string{"24:00", "09:60", "bad"} {
		if _, _, err := parseScheduleTime(value); err == nil {
			t.Fatalf("expected error for %q", value)
		}
	}
}

func TestBuildLaunchAgentPlist(t *testing.T) {
	plist := buildLaunchAgentPlist(0, 5, []string{"/tmp/tokenheat", "run", "daily"}, "/tmp/repo")
	for _, needle := range []string{
		"<string>com.tokenheat.daily-sync</string>",
		"<string>/tmp/tokenheat</string>",
		"<integer>0</integer>",
		"<integer>5</integer>",
		"<string>/tmp/repo</string>",
	} {
		if !strings.Contains(plist, needle) {
			t.Fatalf("plist missing %q", needle)
		}
	}
}
