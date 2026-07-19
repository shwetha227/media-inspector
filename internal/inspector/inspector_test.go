package inspector

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"
)

// testdataDir resolves testdata/ relative to this test file, regardless
// of the working directory `go test` is invoked from.
func testdataDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata")
}

func TestInspect_HappyPath(t *testing.T) {
	path := filepath.Join(testdataDir(t), "sample.mp4")

	info, err := Inspect(path)
	if err != nil {
		t.Fatalf("Inspect(%s) returned error: %v", path, err)
	}

	if info.Container == "" {
		t.Error("expected a non-empty container")
	}
	if len(info.Streams) == 0 {
		t.Fatal("expected at least one stream")
	}
	if info.Streams[0].Type != "video" {
		t.Errorf("Streams[0].Type = %q, want %q", info.Streams[0].Type, "video")
	}
	if info.Streams[0].Width != 320 || info.Streams[0].Height != 240 {
		t.Errorf("unexpected resolution: got %dx%d, want 320x240",
			info.Streams[0].Width, info.Streams[0].Height)
	}
}

func TestInspect_NonExistentPath(t *testing.T) {
	path := filepath.Join(testdataDir(t), "does-not-exist.mp4")

	_, err := Inspect(path)
	if err == nil {
		t.Fatal("expected an error for a non-existent path, got nil")
	}
	if !errors.Is(err, ErrInspectFailed) {
		t.Errorf("expected error to wrap ErrInspectFailed, got: %v", err)
	}
}

func TestInspect_NotMediaFile(t *testing.T) {
	path := filepath.Join(testdataDir(t), "not_media.txt")

	_, err := Inspect(path)
	if err == nil {
		t.Fatal("expected an error for a non-media file, got nil")
	}
}

func TestInspect_EmptyPath(t *testing.T) {
	_, err := Inspect("")
	if err == nil {
		t.Fatal("expected an error for an empty path, got nil")
	}
}