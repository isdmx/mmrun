package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestFileDownload_WritesToDir(t *testing.T) {
	dir := t.TempDir()
	app := &appContext{
		api: &fakeAPI{
			fileInfos: []*model.FileInfo{{Id: "f1", Name: "report.txt"}},
			fileData:  []byte("hello world"),
		},
		outputMode: "ai",
	}
	paths, err := runFileDownload(app, "p1", dir)
	if err != nil {
		t.Fatalf("runFileDownload: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 file, got %d", len(paths))
	}
	got, err := os.ReadFile(filepath.Join(dir, "report.txt"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("content = %q", string(got))
	}
}
