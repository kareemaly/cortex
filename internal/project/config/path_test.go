package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePath(t *testing.T) {
	projectRoot := t.TempDir()

	t.Run("empty path returns empty", func(t *testing.T) {
		result, err := ResolvePath("", projectRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("absolute path returned as-is", func(t *testing.T) {
		absPath := "/some/absolute/path"
		result, err := ResolvePath(absPath, projectRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != absPath {
			t.Errorf("expected %q, got %q", absPath, result)
		}
	})

	t.Run("tilde path expands to home", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("failed to get home dir: %v", err)
		}

		result, err := ResolvePath("~/.cortex/defaults/basic", projectRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := filepath.Join(homeDir, ".cortex", "defaults", "basic")
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("relative path resolved from project root", func(t *testing.T) {
		result, err := ResolvePath("../shared/config", projectRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := filepath.Clean(filepath.Join(projectRoot, "..", "shared", "config"))
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("cleans path with dots", func(t *testing.T) {
		result, err := ResolvePath("/a/b/../c/./d", projectRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "/a/c/d"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestValidateExtendPath(t *testing.T) {
	projectRoot := t.TempDir()

	t.Run("empty path returns empty", func(t *testing.T) {
		result, err := ValidateExtendPath("", projectRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("valid directory returns resolved path", func(t *testing.T) {
		baseDir := filepath.Join(projectRoot, "base-config")
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			t.Fatalf("failed to create base dir: %v", err)
		}

		result, err := ValidateExtendPath(baseDir, projectRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != baseDir {
			t.Errorf("expected %q, got %q", baseDir, result)
		}
	})

	t.Run("relative path resolved and validated", func(t *testing.T) {
		baseDir := filepath.Join(projectRoot, "configs", "base")
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			t.Fatalf("failed to create base dir: %v", err)
		}

		result, err := ValidateExtendPath("configs/base", projectRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != baseDir {
			t.Errorf("expected %q, got %q", baseDir, result)
		}
	})

	t.Run("non-existent path returns error", func(t *testing.T) {
		_, err := ValidateExtendPath("/does/not/exist", projectRoot)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !IsExtendPathNotFound(err) {
			t.Errorf("expected ExtendPathNotFoundError, got %T: %v", err, err)
		}
	})

	t.Run("file path (not directory) returns error", func(t *testing.T) {
		filePath := filepath.Join(projectRoot, "not-a-dir.txt")
		if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		_, err := ValidateExtendPath(filePath, projectRoot)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !IsExtendPathNotFound(err) {
			t.Errorf("expected ExtendPathNotFoundError, got %T: %v", err, err)
		}
	})
}
