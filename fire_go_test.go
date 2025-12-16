package main

import (
	"strings"
	"testing"
)

func TestParseGitLogToTicker_NoOutput(t *testing.T) {
	if _, _, ok := parseGitLogToTicker(""); ok {
		t.Fatalf("expected ok=false for empty log")
	}
}

func TestParseGitLogToTicker_SingleCommit(t *testing.T) {
	log := "abcd1234\tAlice\t3 days ago\tInitial commit"
	msg, meta, ok := parseGitLogToTicker(log)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if msg == "" || meta == "" {
		t.Fatalf("expected non-empty message and meta")
	}
	if got, want := msg, "Initial commit"; !contains(got, want) {
		t.Fatalf("message %q does not contain %q", got, want)
	}
	if got, want := meta, "by Alice 3 days ago"; !contains(got, want) {
		t.Fatalf("meta %q does not contain %q", got, want)
	}
}

func TestParseGitLogToTicker_MultipleCommits(t *testing.T) {
	log := "" +
		"abcd1234\tAlice\t3 days ago\tInitial commit\n" +
		"efgh5678\tBob\t2 weeks ago\tAdd feature X\n" +
		"ijkl9012\tCarol\t1 year ago\tRefactor module Y\n"
	msg, meta, ok := parseGitLogToTicker(log)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	for _, want := range []string{"Initial commit", "Add feature X", "Refactor module Y"} {
		if !contains(msg, want) {
			t.Fatalf("message ticker %q does not contain %q", msg, want)
		}
	}
	for _, want := range []string{"by Alice 3 days ago", "by Bob 2 weeks ago", "by Carol 1 year ago"} {
		if !contains(meta, want) {
			t.Fatalf("meta ticker %q does not contain %q", meta, want)
		}
	}
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}
