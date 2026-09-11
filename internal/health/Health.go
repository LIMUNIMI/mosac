package health

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"mosac/internal/utils"
)

// runs all diagnostics: OS dependencies, Docker status, and Projucer builds.
func Check() error {
	fmt.Println("Running MOSAC system state checks...")

	// xcodebuild (MacOS only)
	if runtime.GOOS == "darwin" {
		fmt.Print("    - Checking xcodebuild dependency... ")
		if _, err := exec.LookPath("xcodebuild"); err != nil {
			fmt.Println("FAILED")
			return fmt.Errorf("xcodebuild not found. Please install Xcode Command Line Tools.")
		}
		fmt.Println("OK")
	} else {
		fmt.Println("    - Host OS is not macOS. Skipping xcodebuild check.")
	}

	// Docker Daemon
	fmt.Println("    - Checking Docker daemon status...")
	cli, err := utils.StartDocker()
	if err != nil {
		return fmt.Errorf("failed to verify or start Docker: %v", err)
	}

	fmt.Println("    - Building required Docker images...")
	err = utils.BuildImageFromEmbedded(cli, "juce-builder:linux", "go_juce_builder_linux")
	err = utils.BuildImageFromEmbedded(cli, "juce-builder:windows", "go_juce_builder_windows")
	if err != nil {
		fmt.Printf("        Error occurred while building Docker images: %v\n", err)
	}

	// indexed JUCE versions and build Projucer if missing
	fmt.Println("    - Checking indexed JUCE versions...")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home directory: %v", err)
	}

	juceRootDir := filepath.Join(homeDir, ".mosac", "juce")
	entries, err := os.ReadDir(juceRootDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("      [!] No JUCE versions found. Use 'mosac update' to download a framework version.")
			return nil
		}
		return fmt.Errorf("error reading JUCE directory: %v", err)
	}

	foundJuce := false
	for _, entry := range entries {
		if entry.IsDir() {
			foundJuce = true
			majorVersion := entry.Name()
			jucePath := filepath.Join(juceRootDir, majorVersion)

			fmt.Printf("\n      > Found JUCE major version '%s' at: %s\n", majorVersion, jucePath)

			if runtime.GOOS == "darwin" {
				projucerPath, err := utils.CheckIfProjucerIsAlreadyBuilt(jucePath)
				if err != nil {
					return fmt.Errorf("failed to verify or build Projucer for JUCE %s: %v", majorVersion, err)
				}
				fmt.Printf("        Projucer is ready at: %s\n", projucerPath)
			} else {
				fmt.Println("        Skipping Projucer build (host is not macOS).")
			}
		}
	}

	if !foundJuce {
		fmt.Println("      [!] No JUCE directories found inside .mosac/juce.")
	}

	fmt.Println("\n    All system checks passed successfully!")
	return nil
}
