package cmd

import (
	"reflect"
	"testing"
)

func TestResolveColumns(t *testing.T) {
	def := []string{"time", "user", "message"}

	got, err := resolveColumns(def, "")
	if err != nil || !reflect.DeepEqual(got, def) {
		t.Errorf("empty: %v %v", got, err)
	}

	got, err = resolveColumns(def, "user,message")
	if err != nil || !reflect.DeepEqual(got, []string{"user", "message"}) {
		t.Errorf("replace: %v %v", got, err)
	}

	got, err = resolveColumns(def, "-time")
	if err != nil || !reflect.DeepEqual(got, []string{"user", "message"}) {
		t.Errorf("remove: %v %v", got, err)
	}

	got, err = resolveColumns(def, "+time")
	if err != nil || !reflect.DeepEqual(got, []string{"time", "user", "message"}) {
		t.Errorf("add existing (no dup): %v %v", got, err)
	}

	if _, err := resolveColumns(def, "bogus"); err == nil {
		t.Error("expected unknown-column error")
	}

	if _, err := resolveColumns(def, "user,-time"); err == nil {
		t.Error("expected mixing error")
	}

	if _, err := resolveColumns(def, "+bogus"); err == nil {
		t.Error("expected unknown-column error for +bogus modifier")
	}
}
