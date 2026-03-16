package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/f1bonacc1/process-compose/src/config"
)

const (
	downloadURLTemplate  = "https://github.com/f1bonacc1/process-compose/releases/download/%s/%s"
	checksumsURLTemplate = "https://github.com/f1bonacc1/process-compose/releases/download/%s/process-compose_checksums.txt"
	binaryName           = "process-compose"
)

func getArchiveName() string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("%s_%s_%s%s", binaryName, runtime.GOOS, runtime.GOARCH, ext)
}

func getDownloadURL(version string) string {
	return fmt.Sprintf(downloadURLTemplate, version, getArchiveName())
}

func getChecksumsURL(version string) string {
	return fmt.Sprintf(checksumsURLTemplate, version)
}

func downloadFile(url string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download %s: HTTP %d", url, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func verifyChecksum(archiveData []byte, archiveName string, checksumsData []byte) error {
	hash := sha256.Sum256(archiveData)
	actualSum := hex.EncodeToString(hash[:])

	lines := strings.Split(string(checksumsData), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == archiveName {
			if fields[0] == actualSum {
				return nil
			}
			return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", archiveName, fields[0], actualSum)
		}
	}
	return fmt.Errorf("checksum not found for %s in checksums file", archiveName)
}

func extractBinary(archiveData []byte, isZip bool) ([]byte, error) {
	targetName := binaryName
	if runtime.GOOS == "windows" {
		targetName = binaryName + ".exe"
	}

	if isZip {
		return extractFromZip(archiveData, targetName)
	}
	return extractFromTarGz(archiveData, targetName)
}

func extractFromTarGz(data []byte, targetName string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar: %w", err)
		}
		if filepath.Base(hdr.Name) == targetName {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("%s not found in archive", targetName)
}

func extractFromZip(data []byte, targetName string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to read zip: %w", err)
	}
	for _, f := range r.File {
		if filepath.Base(f.Name) == targetName {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("%s not found in archive", targetName)
}

// checkCanReplace verifies that the current binary can be replaced before downloading.
func CheckCanReplace() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to find executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}
	if isPackageManaged(execPath) {
		return "", fmt.Errorf("binary at %s appears to be managed by a package manager; please update via your package manager instead", execPath)
	}
	dir := filepath.Dir(execPath)
	tmpFile, err := os.CreateTemp(dir, ".process-compose-update-check-*")
	if err != nil {
		if runtime.GOOS == "windows" {
			return "", fmt.Errorf("no write permission to %s (try running as Administrator): %w", dir, err)
		}
		return "", fmt.Errorf("no write permission to %s (try running with sudo): %w", dir, err)
	}
	tmpFile.Close()
	os.Remove(tmpFile.Name())
	return execPath, nil
}

func replaceBinary(execPath string, newBinary []byte) error {
	// Get existing file info for permissions
	info, err := os.Stat(execPath)
	if err != nil {
		return fmt.Errorf("failed to stat current binary: %w", err)
	}

	dir := filepath.Dir(execPath)
	tmpFile, err := os.CreateTemp(dir, "process-compose-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file (do you have write permission to %s?): %w", dir, err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(newBinary); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write new binary: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpPath, info.Mode()); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if runtime.GOOS == "windows" {
		// Windows can't replace a running binary; rename old one first
		oldPath := execPath + ".old"
		if err := os.Rename(execPath, oldPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to rename old binary: %w", err)
		}
		if err := os.Rename(tmpPath, execPath); err != nil {
			// Try to restore
			_ = os.Rename(oldPath, execPath)
			os.Remove(tmpPath)
			return fmt.Errorf("failed to replace binary: %w", err)
		}
		// Best-effort cleanup
		os.Remove(oldPath)
	} else {
		if err := os.Rename(tmpPath, execPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to replace binary: %w", err)
		}
	}

	return nil
}

func isPackageManaged(path string) bool {
	managedPrefixes := []string{
		"/nix/store",
		"/Cellar/",
		"/cellar/",
		"/snap/",
	}
	for _, prefix := range managedPrefixes {
		if strings.Contains(path, prefix) {
			return true
		}
	}
	return false
}

// Update downloads and installs the latest version of process-compose.
func Update() error {
	execPath, err := CheckCanReplace()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Checking for latest version...")

	latest, err := GetLatestReleaseName()
	if err != nil {
		return fmt.Errorf("failed to check latest version: %w", err)
	}
	fmt.Fprintf(os.Stderr, " \033[32m\u2713\033[0m\n")

	if CompareVersions(config.Version, latest) >= 0 {
		fmt.Fprintf(os.Stderr, "Already up to date (%s)\n", config.Version)
		return nil
	}

	fmt.Fprintf(os.Stderr, "Updating %s -> %s\n", config.Version, latest)

	archiveName := getArchiveName()
	archiveURL := getDownloadURL(latest)
	checksumsURL := getChecksumsURL(latest)

	fmt.Fprintf(os.Stderr, "Downloading %s...", archiveName)
	archiveData, err := downloadFile(archiveURL)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, " \033[32m\u2713\033[0m\n")

	fmt.Fprintf(os.Stderr, "Verifying checksum...")
	checksumsData, err := downloadFile(checksumsURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}

	if err := verifyChecksum(archiveData, archiveName, checksumsData); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, " \033[32m\u2713\033[0m\n")

	fmt.Fprintf(os.Stderr, "Extracting binary...")
	isZip := runtime.GOOS == "windows"
	binaryData, err := extractBinary(archiveData, isZip)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, " \033[32m\u2713\033[0m\n")

	fmt.Fprintf(os.Stderr, "Replacing binary...")
	if err := replaceBinary(execPath, binaryData); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, " \033[32m\u2713\033[0m\n")

	fmt.Fprintf(os.Stderr, "Successfully updated to %s\n", latest)
	return nil
}
