package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestPost_CreatesAndReportsID(t *testing.T) {
	app := &appContext{
		api: &fakeAPI{
			resolved: &model.Channel{Id: "c1"},
			created:  &model.Post{Id: "p1", Message: "hello"},
		},
		outputMode: "ai",
	}
	var buf bytes.Buffer
	err := runPost(app, "eng/general", "hello", postOpts{}, &buf)
	if err != nil {
		t.Fatalf("runPost: %v", err)
	}
	if !strings.Contains(buf.String(), "p1") {
		t.Errorf("expected post id in output:\n%s", buf.String())
	}
}

func TestPost_AttachesMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.txt")
	f2 := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(f1, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}

	fake := &fakeAPI{
		resolved:   &model.Channel{Id: "c1"},
		created:    &model.Post{Id: "p1"},
		uploadResp: &model.FileUploadResponse{FileInfos: []*model.FileInfo{{Id: "f-uploaded"}}},
	}
	app := &appContext{api: fake, outputMode: "ai"}

	var buf bytes.Buffer
	if err := runPost(app, "eng/general", "with files", postOpts{files: []string{f1, f2}}, &buf); err != nil {
		t.Fatalf("runPost: %v", err)
	}
	if fake.lastPost == nil {
		t.Fatal("no post was created")
	}
	if len(fake.lastPost.FileIds) != 2 {
		t.Errorf("expected 2 attached file IDs (one per uploaded file), got %d: %v",
			len(fake.lastPost.FileIds), fake.lastPost.FileIds)
	}
}

func TestPost_Stdin(t *testing.T) {
	fake := &fakeAPI{resolved: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen}, created: &model.Post{Id: "p1"}}
	app := &appContext{api: fake, outputMode: "ai"}
	old := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() { os.Stdin = old }()
	go func() { _, _ = w.Write([]byte("from pipe")); w.Close() }()
	var buf bytes.Buffer
	if err := runPost(app, "eng/general", "-", postOpts{}, &buf); err != nil {
		t.Fatalf("runPost stdin: %v", err)
	}
	if fake.lastPost == nil || fake.lastPost.Message != "from pipe" {
		t.Errorf("stdin message not posted: %+v", fake.lastPost)
	}
}

func TestPost_Editor(t *testing.T) {
	t.Setenv("EDITOR", "cat")
	fake := &fakeAPI{resolved: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen}, created: &model.Post{Id: "p1"}}
	app := &appContext{api: fake, outputMode: "ai"}
	var buf bytes.Buffer
	if err := runPost(app, "eng/general", "prefill", postOpts{editor: true}, &buf); err != nil {
		t.Fatalf("runPost editor: %v", err)
	}
	if fake.lastPost == nil || !strings.Contains(fake.lastPost.Message, "prefill") {
		t.Errorf("editor message not posted: %+v", fake.lastPost)
	}
}

func TestPost_DryRun(t *testing.T) {
	fake := &fakeAPI{resolved: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen}}
	app := &appContext{api: fake, outputMode: "ai"}
	var buf bytes.Buffer
	if err := runPost(app, "eng/general", "hello", postOpts{dryRun: true}, &buf); err != nil {
		t.Fatalf("runPost dry-run: %v", err)
	}
	if fake.lastPost != nil {
		t.Error("dry-run must not create a post")
	}
	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("dry-run should preview the message:\n%s", buf.String())
	}
}
