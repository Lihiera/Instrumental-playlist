package app

import "testing"

func TestRunAcceptsNoArgs(t *testing.T) {
	if err := Run(nil); err != nil {
		t.Fatalf("Run(nil) returned error: %v", err)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	if err := Run([]string{"convert"}); err == nil {
		t.Fatal("Run returned nil for an unimplemented command")
	}
}
