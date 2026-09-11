package juce

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

// Update downloads and installs the specified JUCE version into ~/.mosac/juce/<major_version>
func Update(version string) {
	fmt.Printf("    Checking requested JUCE version: %s\n", version)

	// resolve "latest" by querying the GitHub API
	if version == "latest" {
		latestTag, err := getLatestJuceVersion()
		if err != nil {
			fmt.Printf("Error fetching latest version from GitHub: %v\n", err)
			os.Exit(1)
		}
		version = latestTag
		fmt.Printf("    Latest version resolved to: %s\n", version)
	}

	// determine the Major version (e.g., "8.0.4" -> "8")
	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		fmt.Println("Error: Invalid version format.")
		os.Exit(1)
	}
	majorVersion := parts[0]

	// prepare destination paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting user home directory: %v\n", err)
		os.Exit(1)
	}
	mosacJuceDir := filepath.Join(homeDir, ".mosac", "juce", majorVersion)

	fmt.Printf("    Target install directory: %s\n", mosacJuceDir)

	// if the major version directory exists, remove it to replace it
	if _, err := os.Stat(mosacJuceDir); !os.IsNotExist(err) {
		fmt.Printf("    Found existing JUCE %s.x installation. Removing it...\n", majorVersion)
		os.RemoveAll(mosacJuceDir)
	}

	// download the ZIP file from GitHub
	zipURL := fmt.Sprintf("https://github.com/juce-framework/JUCE/archive/refs/tags/%s.zip", version)
	fmt.Printf("    Downloading from: %s\n", zipURL)

	tempZipPath := filepath.Join(os.TempDir(), fmt.Sprintf("juce-%s.zip", version))
	err = downloadFile(tempZipPath, zipURL)
	if err != nil {
		fmt.Printf("Error downloading JUCE: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tempZipPath)

	// extract the ZIP file
	fmt.Println("    Extracting framework...")
	err = unzipAndStripRoot(tempZipPath, mosacJuceDir)
	if err != nil {
		fmt.Printf("Error extracting JUCE: %v\n", err)
		os.Exit(1)
	}

	// write the .version file to keep track of the exact installed version
	versionFilePath := filepath.Join(mosacJuceDir, ".version")
	err = os.WriteFile(versionFilePath, []byte(version), 0644)
	if err != nil {
		fmt.Printf("Warning: Could not write .version file: %v\n", err)
	}

	fmt.Printf("    Success! JUCE %s installed in %s\n", version, mosacJuceDir)
}

// contacts GitHub to find the latest JUCE release tag
func getLatestJuceVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/juce-framework/JUCE/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

// downloads a file from a URL and saves it to the disk
func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download, status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// extracts the ZIP file while stripping the root folder so that its contents are placed directly in targetDir.
func unzipAndStripRoot(zipPath, targetDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	os.MkdirAll(targetDir, 0755)

	for _, f := range r.File {
		parts := strings.Split(filepath.ToSlash(f.Name), "/")
		if len(parts) <= 1 {
			continue
		}

		strippedPath := strings.Join(parts[1:], "/")
		fpath := filepath.Join(targetDir, strippedPath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
