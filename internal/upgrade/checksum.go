package upgrade

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// DownloadTimeout is the timeout for downloading files.
	DownloadTimeout = 5 * time.Minute
)

// ParseChecksums parses a checksums.txt file in the format "<sha256>  <filename>".
// Returns a map of filename -> checksum.
func ParseChecksums(content string) map[string]string {
	checksums := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Format: "<sha256>  <filename>" (two spaces between hash and filename)
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			// Try single space as fallback
			parts = strings.SplitN(line, " ", 2)
			if len(parts) != 2 {
				continue
			}
		}

		checksum := strings.TrimSpace(parts[0])
		filename := strings.TrimSpace(parts[1])
		if checksum != "" && filename != "" {
			checksums[filename] = checksum
		}
	}

	return checksums
}

// VerifyChecksum computes the SHA256 checksum of a file and compares it to the expected value.
func VerifyChecksum(filepath, expected string) error {
	f, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}

// Download fetches a URL and saves it to a file.
func Download(url, destPath string) error {
	client := &http.Client{Timeout: DownloadTimeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "cortex-upgrade")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// DownloadString fetches a URL and returns its content as a string.
func DownloadString(url string) (string, error) {
	client := &http.Client{Timeout: GitHubAPITimeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "cortex-upgrade")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(content), nil
}
