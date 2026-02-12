package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type VersionManager struct {
	dataDir string
}

type GithubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GithubAsset `json:"assets"`
}

type GithubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func NewVersionManager(dataDir string) *VersionManager {
	frpcDir := filepath.Join(dataDir, "frpc")
	os.MkdirAll(frpcDir, 0755)
	return &VersionManager{dataDir: dataDir}
}

func (vm *VersionManager) frpcPath() string {
	name := "frpc"
	if runtime.GOOS == "windows" {
		name = "frpc.exe"
	}
	return filepath.Join(vm.dataDir, "frpc", name)
}

// frpcBinaryName returns "frpc" or "frpc.exe" depending on OS
func frpcBinaryName() string {
	if runtime.GOOS == "windows" {
		return "frpc.exe"
	}
	return "frpc"
}

// GetCurrentVersion returns the installed frpc version
func (vm *VersionManager) GetCurrentVersion() (string, error) {
	frpcPath := vm.frpcPath()
	if _, err := os.Stat(frpcPath); err != nil {
		return "", fmt.Errorf("frpc not installed")
	}

	cmd := exec.Command(frpcPath, "--version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version: %v", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// GetLatestRelease fetches the latest release info from GitHub
func (vm *VersionManager) GetLatestRelease() (*GithubRelease, error) {
	resp, err := http.Get("https://api.github.com/repos/fatedier/frp/releases/latest")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from GitHub: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &release, nil
}

// assetPattern returns the expected filename pattern for the current OS/arch
func assetPattern(version string) (pattern string, isZip bool) {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	ver := strings.TrimPrefix(version, "v")

	if osName == "windows" {
		return fmt.Sprintf("frp_%s_%s_%s.zip", ver, osName, arch), true
	}
	return fmt.Sprintf("frp_%s_%s_%s.tar.gz", ver, osName, arch), false
}

// InstallFromGitHub downloads and installs the latest frpc from GitHub
func (vm *VersionManager) InstallFromGitHub() (string, error) {
	release, err := vm.GetLatestRelease()
	if err != nil {
		return "", err
	}

	pattern, _ := assetPattern(release.TagName)

	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == pattern {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return "", fmt.Errorf("no matching asset found for %s/%s (looking for %s)", runtime.GOOS, runtime.GOARCH, pattern)
	}

	log.Printf("Downloading frpc from: %s", downloadURL)

	// Download
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("download failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Save to temp file
	tmpFile := filepath.Join(vm.dataDir, "frpc_download"+filepath.Ext(pattern))
	f, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return "", err
	}
	f.Close()

	// Extract
	if err := vm.extractFrpc(tmpFile); err != nil {
		return "", err
	}

	// Clean up
	os.Remove(tmpFile)

	version, _ := vm.GetCurrentVersion()
	log.Printf("frpc installed successfully: %s", version)
	return version, nil
}

// InstallFromUpload installs frpc from an uploaded archive file
func (vm *VersionManager) InstallFromUpload(reader io.Reader) (string, error) {
	// Detect format by trying to save then extract
	tmpFile := filepath.Join(vm.dataDir, "frpc_upload")
	f, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(f, reader); err != nil {
		f.Close()
		return "", err
	}
	f.Close()

	if err := vm.extractFrpc(tmpFile); err != nil {
		os.Remove(tmpFile)
		return "", err
	}

	os.Remove(tmpFile)

	version, _ := vm.GetCurrentVersion()
	log.Printf("frpc installed from upload: %s", version)
	return version, nil
}

// extractFrpc extracts the frpc binary from an archive (tar.gz or zip)
func (vm *VersionManager) extractFrpc(archivePath string) error {
	// Try zip first (for Windows packages)
	if err := vm.extractFromZip(archivePath); err == nil {
		return nil
	}

	// Try tar.gz (for Linux packages)
	if err := vm.extractFromTarGz(archivePath); err == nil {
		return nil
	}

	return fmt.Errorf("failed to extract frpc: unsupported archive format")
}

// extractFromZip extracts frpc binary from a .zip archive
func (vm *VersionManager) extractFromZip(archivePath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	binName := frpcBinaryName()

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name == binName && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			dst := vm.frpcPath()
			outFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, rc); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

			log.Printf("Extracted %s to %s (from zip)", binName, dst)
			return nil
		}
	}

	return fmt.Errorf("frpc binary not found in zip archive")
}

// extractFromTarGz extracts frpc binary from a .tar.gz archive
func (vm *VersionManager) extractFromTarGz(archivePath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip open failed: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	binName := frpcBinaryName()

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %v", err)
		}

		// Look for the frpc binary (not frps)
		name := filepath.Base(header.Name)
		if name == binName && header.Typeflag == tar.TypeReg {
			dst := vm.frpcPath()
			outFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

			log.Printf("Extracted %s to %s (from tar.gz)", binName, dst)
			return nil
		}
	}

	return fmt.Errorf("frpc binary not found in tar.gz archive")
}

// IsInstalled checks if frpc binary exists
func (vm *VersionManager) IsInstalled() bool {
	_, err := os.Stat(vm.frpcPath())
	return err == nil
}
