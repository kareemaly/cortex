package upgrade

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// --- Version comparison (table-driven) ---

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.1.0", "1.2.0", -1},
		{"1.0.1", "1.0.2", -1},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.0", "v1.0.0", 0},
		{"v1.0.0", "v2.0.0", -1},
		// Prerelease stripped
		{"1.0.0-rc1", "1.0.0", 0},
		{"1.0.0-beta", "1.0.0-rc1", 0},
		// Partial versions
		{"1.0", "1.0.0", 0},
		{"1", "1.0.0", 0},
		// Minor differences
		{"0.9.0", "0.10.0", -1},
		{"0.10.0", "0.9.0", 1},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tt.a, tt.b), func(t *testing.T) {
			got := compareVersions(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestShouldUpgrade(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
		want    bool
	}{
		{"dev always upgrades", "dev", "v1.0.0", true},
		{"same version no upgrade", "v1.0.0", "v1.0.0", false},
		{"newer available", "v1.0.0", "v1.1.0", true},
		{"older no upgrade", "v1.1.0", "v1.0.0", false},
		{"major upgrade", "v1.0.0", "v2.0.0", true},
		{"patch upgrade", "v1.0.0", "v1.0.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldUpgrade(tt.current, tt.target)
			if got != tt.want {
				t.Errorf("shouldUpgrade(%q, %q) = %v, want %v", tt.current, tt.target, got, tt.want)
			}
		})
	}
}

// --- Checksum (table-driven + filesystem) ---

func TestParseChecksums(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]string
	}{
		{
			name:    "two-space format",
			content: "abc123  cortex-darwin-arm64\ndef456  cortexd-darwin-arm64\n",
			want:    map[string]string{"cortex-darwin-arm64": "abc123", "cortexd-darwin-arm64": "def456"},
		},
		{
			name:    "single-space fallback",
			content: "abc123 cortex-linux-amd64\n",
			want:    map[string]string{"cortex-linux-amd64": "abc123"},
		},
		{
			name:    "empty lines ignored",
			content: "abc123  file1\n\n\ndef456  file2\n",
			want:    map[string]string{"file1": "abc123", "file2": "def456"},
		},
		{
			name:    "empty content",
			content: "",
			want:    map[string]string{},
		},
		{
			name:    "malformed lines skipped",
			content: "malformed_no_space\nabc123  file1\n",
			want:    map[string]string{"file1": "abc123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseChecksums(tt.content)
			if len(got) != len(tt.want) {
				t.Errorf("len = %d, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("got[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestVerifyChecksum_Success(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "testfile")
	content := []byte("hello world")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatal(err)
	}

	h := sha256.Sum256(content)
	expected := hex.EncodeToString(h[:])

	if err := VerifyChecksum(filePath, expected); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestVerifyChecksum_Mismatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "testfile")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	err := VerifyChecksum(filePath, "wronghash")
	if err == nil {
		t.Fatal("expected error for mismatched checksum")
	}
}

func TestVerifyChecksum_FileNotFound(t *testing.T) {
	err := VerifyChecksum("/nonexistent/path/file", "abc123")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// --- Download (httptest) ---

func TestDownload_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("binary content"))
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "download")

	if err := Download(srv.URL+"/file", destPath); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "binary content" {
		t.Errorf("expected 'binary content', got %q", string(content))
	}
}

func TestDownload_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "download")

	err := Download(srv.URL+"/file", destPath)
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestDownloadString_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello string"))
	}))
	defer srv.Close()

	content, err := DownloadString(srv.URL + "/file")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if content != "hello string" {
		t.Errorf("expected 'hello string', got %q", content)
	}
}

func TestDownloadString_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := DownloadString(srv.URL + "/missing")
	if err == nil {
		t.Fatal("expected error on 404")
	}
}

// --- GitHub (pure functions) ---

func TestGetAssetURL_Found(t *testing.T) {
	release := &Release{
		TagName: "v1.0.0",
		Assets: []Asset{
			{Name: "cortex-darwin-arm64", BrowserDownloadURL: "https://example.com/cortex-darwin-arm64"},
			{Name: "cortexd-darwin-arm64", BrowserDownloadURL: "https://example.com/cortexd-darwin-arm64"},
		},
	}

	url, err := GetAssetURL(release, "cortex-darwin-arm64")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://example.com/cortex-darwin-arm64" {
		t.Errorf("unexpected URL: %q", url)
	}
}

func TestGetAssetURL_NotFound(t *testing.T) {
	release := &Release{
		TagName: "v1.0.0",
		Assets:  []Asset{{Name: "other"}},
	}

	_, err := GetAssetURL(release, "cortex-linux-amd64")
	if err == nil {
		t.Fatal("expected error for missing asset")
	}
}

func TestGetDownloadBaseURL(t *testing.T) {
	url := GetDownloadBaseURL("v1.2.3")
	expected := fmt.Sprintf("https://github.com/%s/%s/releases/download/v1.2.3", GitHubOwner, GitHubRepo)
	if url != expected {
		t.Errorf("expected %q, got %q", expected, url)
	}
}

// --- Binary (filesystem) ---

func TestGetBinaryName(t *testing.T) {
	name := GetBinaryName("cortex")
	expected := fmt.Sprintf("cortex-%s-%s", runtime.GOOS, runtime.GOARCH)
	if name != expected {
		t.Errorf("expected %q, got %q", expected, name)
	}
}

func TestBackupBinary_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a source "binary"
	srcPath := filepath.Join(tmpDir, "cortex")
	srcContent := []byte("binary data")
	if err := os.WriteFile(srcPath, srcContent, 0755); err != nil {
		t.Fatal(err)
	}

	backupDir := filepath.Join(tmpDir, "backups")
	backupPath, err := BackupBinary(srcPath, backupDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify backup exists and has correct content
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(backupContent) != string(srcContent) {
		t.Error("backup content does not match source")
	}

	// Verify naming includes "cortex." and ".bak"
	base := filepath.Base(backupPath)
	if len(base) < 12 || base[:7] != "cortex." || base[len(base)-4:] != ".bak" {
		t.Errorf("unexpected backup filename: %q", base)
	}
}

func TestBackupBinary_SourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := BackupBinary(filepath.Join(tmpDir, "nonexistent"), filepath.Join(tmpDir, "backups"))
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestReplaceBinary_NoSudo(t *testing.T) {
	tmpDir := t.TempDir()

	oldPath := filepath.Join(tmpDir, "old")
	newPath := filepath.Join(tmpDir, "new")

	if err := os.WriteFile(oldPath, []byte("old content"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("new content"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := ReplaceBinary(oldPath, newPath, false); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	content, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new content" {
		t.Errorf("expected 'new content', got %q", string(content))
	}
}

func TestCleanupBackups_KeepsRecent(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create 5 cortex backups with ordered timestamps
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("cortex.20250101-10000%d.bak", i)
		if err := os.WriteFile(filepath.Join(backupDir, name), []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := CleanupBackups(backupDir, 3); err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(backupDir)
	if len(entries) != 3 {
		t.Errorf("expected 3 remaining, got %d", len(entries))
	}

	// Verify oldest were removed
	for _, e := range entries {
		name := e.Name()
		// Should keep 02, 03, 04 (the 3 newest)
		if name == "cortex.20250101-100000.bak" || name == "cortex.20250101-100001.bak" {
			t.Errorf("expected oldest backup %q to be removed", name)
		}
	}
}

func TestCleanupBackups_NothingToClean(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create 2 backups (fewer than keepCount=3)
	for i := 0; i < 2; i++ {
		name := fmt.Sprintf("cortex.20250101-10000%d.bak", i)
		if err := os.WriteFile(filepath.Join(backupDir, name), []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := CleanupBackups(backupDir, 3); err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(backupDir)
	if len(entries) != 2 {
		t.Errorf("expected 2 remaining, got %d", len(entries))
	}
}

func TestCleanupBackups_NonexistentDir(t *testing.T) {
	err := CleanupBackups("/nonexistent/dir/backups", 3)
	if err != nil {
		t.Errorf("expected nil error for nonexistent dir, got %v", err)
	}
}

func TestGetBackupDir(t *testing.T) {
	dir, err := GetBackupDir()
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(dir) {
		t.Error("expected absolute path")
	}
	if filepath.Base(dir) != "backups" {
		t.Errorf("expected dir to end with 'backups', got %q", dir)
	}
	parent := filepath.Base(filepath.Dir(dir))
	if parent != ".cortex" {
		t.Errorf("expected parent dir '.cortex', got %q", parent)
	}
}
