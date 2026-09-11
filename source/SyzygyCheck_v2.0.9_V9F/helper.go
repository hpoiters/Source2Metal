package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const tbcheckReleaseURL = "https://raw.githubusercontent.com/hpoiters/Source2Metal/v3.0.7/source/SyzygyCheck_v2.0.7/tbcheck.exe"

type helperHandle struct {
	Path    string
	Origin  string
	cleanup func()
}

func (h helperHandle) Close() {
	if h.cleanup != nil {
		h.cleanup()
	}
}

// acquireTBCheck preserves the checker binary from Ronald de Man unchanged. If the
// exact known helper is already next to SyzygyCheck it is used there. Otherwise
// v2.0.9 downloads the immutable v3.0.7 copy to a temporary directory, verifies
// its SHA-256, uses it for this run, and removes it afterwards.
func acquireTBCheck(appDir string) (helperHandle, error) {
	local := filepath.Join(appDir, "tbcheck.exe")
	if st, err := os.Stat(local); err == nil && !st.IsDir() {
		sum, err := fileSHA256(local)
		if err != nil {
			return helperHandle{}, err
		}
		if !equalFoldASCII(sum, expectedTBCheckSHA256) {
			return helperHandle{}, fmt.Errorf("tbcheck.exe SHA-256 mismatch; expected %s, got %s", expectedTBCheckSHA256, sum)
		}
		return helperHandle{Path: local, Origin: "validated local tbcheck.exe"}, nil
	}

	tempDir, err := os.MkdirTemp("", "SyzygyCheck_AL_tbcheck_")
	if err != nil {
		return helperHandle{}, err
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }
	tempPath := filepath.Join(tempDir, "tbcheck.exe")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Get(tbcheckReleaseURL)
	if err != nil {
		cleanup()
		return helperHandle{}, fmt.Errorf("tbcheck.exe is not present locally and the verified helper could not be downloaded: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		cleanup()
		return helperHandle{}, fmt.Errorf("tbcheck.exe download returned HTTP %s", resp.Status)
	}

	f, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
	if err != nil {
		cleanup()
		return helperHandle{}, err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		cleanup()
		return helperHandle{}, copyErr
	}
	if closeErr != nil {
		cleanup()
		return helperHandle{}, closeErr
	}

	sum, err := fileSHA256(tempPath)
	if err != nil {
		cleanup()
		return helperHandle{}, err
	}
	if !equalFoldASCII(sum, expectedTBCheckSHA256) {
		cleanup()
		return helperHandle{}, fmt.Errorf("downloaded tbcheck.exe SHA-256 mismatch; expected %s, got %s", expectedTBCheckSHA256, sum)
	}

	return helperHandle{Path: tempPath, Origin: "temporary verified tbcheck.exe from Source2Metal v3.0.7", cleanup: cleanup}, nil
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
