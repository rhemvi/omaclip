package clipboard

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestReadCommandOutput_UnderLimit(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), "head", "-c", "100", "/dev/zero")
	data, err := readCommandOutput(cmd, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 100 {
		t.Errorf("got %d bytes, want 100", len(data))
	}
}

func TestReadCommandOutput_OverLimit(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), "head", "-c", "2048", "/dev/zero")
	_, err := readCommandOutput(cmd, 1024)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errOutputTooLarge) {
		t.Errorf("expected errOutputTooLarge, got: %v", err)
	}
}

func TestReadCommandOutput_ExactLimit(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), "head", "-c", "1024", "/dev/zero")
	data, err := readCommandOutput(cmd, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 1024 {
		t.Errorf("got %d bytes, want 1024", len(data))
	}
}

func TestReadCommandOutput_CommandFailure(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), "false")
	_, err := readCommandOutput(cmd, 1024)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
