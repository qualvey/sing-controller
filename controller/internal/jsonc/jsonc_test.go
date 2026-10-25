package jsonc

import (
	"strings"
	"testing"
)

func TestStripCommentsPreservesStringsAndNewlines(t *testing.T) {
	input := []byte("{\n  // line comment\n  \"url\": \"https://example.com/a/*b*/\", /* block\ncomment */\n  \"enabled\": true\n}")
	got, err := StripComments(input)
	if err != nil {
		t.Fatalf("StripComments() error = %v", err)
	}
	text := string(got)
	if strings.Contains(text, "line comment") || strings.Contains(text, "block") {
		t.Fatalf("comments were not removed: %s", text)
	}
	if !strings.Contains(text, "https://example.com/a/*b*/") {
		t.Fatalf("comment-like string content changed: %s", text)
	}
	if strings.Count(text, "\n") != strings.Count(string(input), "\n") {
		t.Fatalf("newlines changed: got %d, want %d", strings.Count(text, "\n"), strings.Count(string(input), "\n"))
	}
}

func TestStripCommentsRejectsUnterminatedBlock(t *testing.T) {
	if _, err := StripComments([]byte(`{"value": 1 /* missing close`)); err == nil {
		t.Fatal("StripComments() expected an error for unterminated block comment")
	}
}
