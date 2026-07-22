package cmd

import (
	"bytes"
	"testing"

	"github.com/isdmx/mmrun/internal/session"
)

func TestContextList(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	_ = session.Save(&session.Session{
		ServerURL: "https://mm.example.com", Token: "t1", UserID: "u1",
		ContextName: "default",
	})
	app := &appContext{api: &fakeAPI{}, outputMode: "ai"}
	var buf bytes.Buffer
	if err := runContextList(app, &buf); err != nil {
		t.Fatalf("context list: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("default")) {
		t.Error("should list the default context")
	}
}

func TestContextCurrent(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	_ = session.Save(&session.Session{ServerURL: "x", Token: "t", UserID: "u1", ContextName: "work"})
	st, _ := session.LoadAll()
	st.Current = "work"
	_ = session.SaveStore(st)
	st2, _ := session.LoadAll()
	if st2.Current != "work" {
		t.Errorf("current = %q, want work", st2.Current)
	}
}
