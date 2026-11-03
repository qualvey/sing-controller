package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindSingBoxBinaryNormalizesConfiguredRelativePath(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "sing-box")
	if err := os.WriteFile(binary, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	previous := BinaryPath
	BinaryPath = binary
	t.Cleanup(func() { BinaryPath = previous })

	got := FindSingBoxBinary()
	want, err := filepath.Abs(binary)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("FindSingBoxBinary() = %q, want %q", got, want)
	}
}
