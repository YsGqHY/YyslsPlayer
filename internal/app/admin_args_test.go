//go:build windows

package app

import "testing"

func TestElevatedParamsFromArgsAddsRelaunchFlagAndSkipsExisting(t *testing.T) {
	got := elevatedParamsFromArgs([]string{"app.exe", "--foo", relaunchFlag, "two words", `quote"me`})
	if got == "" {
		t.Fatal("params should not be empty")
	}
	if count := countToken(got, relaunchFlag); count != 1 {
		t.Fatalf("params = %q, relaunch flag count = %d, want 1", got, count)
	}
	if !containsToken(got, "--foo") || !containsToken(got, `"two words"`) || !containsToken(got, `quote\"me`) {
		t.Fatalf("params = %q missing escaped arguments", got)
	}
}

func TestHasArgInSkipsProgramName(t *testing.T) {
	if hasArgIn([]string{relaunchFlag}, relaunchFlag) {
		t.Fatal("program name should not be considered an argument")
	}
	if !hasArgIn([]string{"app.exe", relaunchFlag}, relaunchFlag) {
		t.Fatal("relaunch flag should be detected in argv[1:]")
	}
}

func TestStripArgKeepsProgramNameAndOrder(t *testing.T) {
	got := stripArg([]string{"app.exe", "a", relaunchFlag, "b", relaunchFlag}, relaunchFlag)
	want := []string{"app.exe", "a", "b"}
	if len(got) != len(want) {
		t.Fatalf("stripArg len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stripArg[%d] = %q, want %q: %#v", i, got[i], want[i], got)
		}
	}
}

func countToken(s, token string) int {
	count := 0
	for start := 0; ; {
		idx := indexFrom(s, token, start)
		if idx < 0 {
			return count
		}
		count++
		start = idx + len(token)
	}
}

func containsToken(s, token string) bool {
	return indexFrom(s, token, 0) >= 0
}

func indexFrom(s, substr string, start int) int {
	if start >= len(s) {
		return -1
	}
	for i := start; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
