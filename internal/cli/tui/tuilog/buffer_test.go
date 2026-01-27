package tuilog

import (
	"fmt"
	"testing"
)

func TestBuffer_BasicLogAndOrder(t *testing.T) {
	buf := NewBuffer(10)

	buf.Info("src", "first")
	buf.Info("src", "second")
	buf.Info("src", "third")

	entries := buf.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Newest first.
	if entries[0].Message != "third" {
		t.Errorf("expected newest first, got %q", entries[0].Message)
	}
	if entries[2].Message != "first" {
		t.Errorf("expected oldest last, got %q", entries[2].Message)
	}
}

func TestBuffer_RingWrap(t *testing.T) {
	buf := NewBuffer(3)

	buf.Info("src", "a")
	buf.Info("src", "b")
	buf.Info("src", "c")
	buf.Info("src", "d") // overwrites "a"

	entries := buf.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries after wrap, got %d", len(entries))
	}

	// Should contain d, c, b (newest first).
	if entries[0].Message != "d" {
		t.Errorf("expected 'd', got %q", entries[0].Message)
	}
	if entries[1].Message != "c" {
		t.Errorf("expected 'c', got %q", entries[1].Message)
	}
	if entries[2].Message != "b" {
		t.Errorf("expected 'b', got %q", entries[2].Message)
	}
}

func TestBuffer_ErrorWarnCounts(t *testing.T) {
	buf := NewBuffer(100)

	buf.Debug("src", "d1")
	buf.Info("src", "i1")
	buf.Warn("src", "w1")
	buf.Error("src", "e1")
	buf.Warn("src", "w2")
	buf.Error("src", "e2")

	if got := buf.ErrorCount(); got != 2 {
		t.Errorf("expected 2 errors, got %d", got)
	}
	if got := buf.WarnCount(); got != 2 {
		t.Errorf("expected 2 warns, got %d", got)
	}
}

func TestBuffer_CountsThroughOverwrites(t *testing.T) {
	buf := NewBuffer(3)

	buf.Error("src", "e1") // slot 0: error
	buf.Warn("src", "w1")  // slot 1: warn
	buf.Info("src", "i1")  // slot 2: info

	if got := buf.ErrorCount(); got != 1 {
		t.Errorf("expected 1 error, got %d", got)
	}
	if got := buf.WarnCount(); got != 1 {
		t.Errorf("expected 1 warn, got %d", got)
	}

	// Overwrite error with info.
	buf.Info("src", "i2") // overwrites e1
	if got := buf.ErrorCount(); got != 0 {
		t.Errorf("expected 0 errors after overwrite, got %d", got)
	}

	// Overwrite warn with error.
	buf.Error("src", "e2") // overwrites w1
	if got := buf.ErrorCount(); got != 1 {
		t.Errorf("expected 1 error, got %d", got)
	}
	if got := buf.WarnCount(); got != 0 {
		t.Errorf("expected 0 warns after overwrite, got %d", got)
	}
}

func TestBuffer_Empty(t *testing.T) {
	buf := NewBuffer(10)

	entries := buf.Entries()
	if entries != nil {
		t.Errorf("expected nil entries for empty buffer, got %v", entries)
	}
	if got := buf.ErrorCount(); got != 0 {
		t.Errorf("expected 0 errors, got %d", got)
	}
	if got := buf.WarnCount(); got != 0 {
		t.Errorf("expected 0 warns, got %d", got)
	}
}

func TestBuffer_Formatf(t *testing.T) {
	buf := NewBuffer(10)

	buf.Debugf("src", "debug %d", 1)
	buf.Infof("src", "info %s", "test")
	buf.Warnf("src", "warn %v", true)
	buf.Errorf("src", "error %d/%d", 1, 2)

	entries := buf.Entries()
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	expected := []string{"error 1/2", "warn true", "info test", "debug 1"}
	for i, e := range entries {
		if e.Message != expected[i] {
			t.Errorf("entry %d: expected %q, got %q", i, expected[i], e.Message)
		}
	}
}

func TestBuffer_LevelStrings(t *testing.T) {
	tests := []struct {
		level Level
		full  string
		short string
	}{
		{LevelDebug, "DEBUG", "D"},
		{LevelInfo, "INFO", "I"},
		{LevelWarn, "WARN", "W"},
		{LevelError, "ERROR", "E"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("Level_%s", tt.full), func(t *testing.T) {
			if got := tt.level.String(); got != tt.full {
				t.Errorf("String() = %q, want %q", got, tt.full)
			}
			if got := tt.level.ShortString(); got != tt.short {
				t.Errorf("ShortString() = %q, want %q", got, tt.short)
			}
		})
	}
}
