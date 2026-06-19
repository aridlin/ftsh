package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestProfilesCommand(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"profiles"}, &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "personal") {
		t.Fatalf("profiles output missing personal: %s", out.String())
	}
}

func TestUnknownCommandErrors(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"nope"}, &out, &out)
	if err == nil {
		t.Fatal("expected unknown command error")
	}
}
