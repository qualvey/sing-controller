package main

import "testing"

func TestWithBuildHash(t *testing.T) {
	tests := []struct {
		name, base, hash, want string
	}{
		{name: "adds short hash", base: "0.11.0", hash: "be7daab2062eac3a", want: "0.11.0-be7daab"},
		{name: "does not duplicate hash", base: "0.11.0-SNAPSHOT-be7daab", hash: "be7daab2062eac3a", want: "0.11.0-SNAPSHOT-be7daab"},
		{name: "keeps dev without metadata", base: "dev", hash: "", want: "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := withBuildHash(tt.base, tt.hash); got != tt.want {
				t.Fatalf("withBuildHash(%q, %q) = %q, want %q", tt.base, tt.hash, got, tt.want)
			}
		})
	}
}
