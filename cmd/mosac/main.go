package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"mosac/internal/juce"
)

func main() {
	// ------- LOCK	------------------------------------------
	lockFile, lockPath, err := acquireLock()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("If you are certain MOSAC is not running, manually delete the file:\n%s\n", lockPath)
		os.Exit(1)
	}
	defer releaseLock(lockFile, lockPath)
	setupSignalHandler(lockFile, lockPath)
	// --------------------------------------------------------

	if len(os.Args) < 2 {
		printRootHelp()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "create-batch":
		createBatchCSVFile(os.Args[2:])
	case "health":
		runHealthCmd()
	case "build":
		runBuildCmd(os.Args[2:])
	case "update":
		runUpdateCmd(os.Args[2:])
	case "-h", "--help", "help":
		printRootHelp()
		os.Exit(0)
	default:
		fmt.Printf("Error: unknown command '%s'.\n\n", os.Args[1])
		printRootHelp()
		os.Exit(1)
	}
}

// -------------------------------------------------------------
// SUBCOMMAND: batch
// -------------------------------------------------------------
func createBatchCSVFile(args []string) {
	batchCmd := flag.NewFlagSet("create-batch", flag.ExitOnError)

	dirPtr := batchCmd.String("dir", "", "directory containing the projects")
	jucePtr := batchCmd.String("juce", "", "default JUCE version to use")
	outPtr := batchCmd.String("out", "", "output directory")
	buildTypePtr := batchCmd.String("c", "Release", "build configuration (Debug or Release)")
	sysPtr := batchCmd.String("sys", "MacOS,Linux,Windows", "comma-separated target system")
	formatsPtr := batchCmd.String("formats", "Standalone,VST3,AU,LV2,Unity,AAX", "comma-separated build formats")

	batchCmd.Parse(args)

	if *dirPtr == "" || *jucePtr == "" || *outPtr == "" {
		fmt.Println("Error: Missing required flags for batch command.")
		batchCmd.Usage()
		os.Exit(1)
	}

	if !IsFlagValid(buildTypePtr, []string{"Debug", "Release"}) {
		fmt.Println("Error: Invalid build configuration. Must be 'Debug' or 'Release'.")
		batchCmd.Usage()
		os.Exit(1)
	}

	if !IsFlagValid(sysPtr, []string{"MacOS", "Linux", "Windows"}) {
		fmt.Println("Error: Invalid system flag. Must be a comma-separated list of 'MacOS', 'Linux', 'Windows'.")
		batchCmd.Usage()
		os.Exit(1)
	}

	if !IsFlagValid(formatsPtr, []string{"Standalone", "VST3", "AU", "LV2", "Unity", "AAX"}) {
		fmt.Println("Error: Invalid formats flag. Must be a comma-separated list of 'Standalone', 'VST3', 'AU', 'LV2', 'Unity', 'AAX'.")
		batchCmd.Usage()
		os.Exit(1)
	}

	//TODO: batch maker logic

}

// -------------------------------------------------------------
// SUBCOMMAND: health
// -------------------------------------------------------------
func runHealthCmd() {
	// checks for Docker installation, Docker daemon running, and JUCE version availability
}

// -------------------------------------------------------------
// SUBCOMMAND: update
// -------------------------------------------------------------
func runUpdateCmd(args []string) {
	updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
	versionPtr := updateCmd.String("v", "latest", "desired JUCE version to update to (default: latest)")

	updateCmd.Parse(args)

	fmt.Printf("Updating JUCE to version: %s\n", *versionPtr)
	juce.Update(*versionPtr)
}

// -------------------------------------------------------------
// SUBCOMMAND: build
// -------------------------------------------------------------
func runBuildCmd(args []string) {
	buildCmd := flag.NewFlagSet("build", flag.ExitOnError)

	batchFilePtr := buildCmd.String("b", "", "path to a batch.csv file")

	ppPtr := buildCmd.String("PP", "", "single project path")
	jpPtr := buildCmd.String("JP", "", "JUCE path")
	opPtr := buildCmd.String("OP", "", "output path for the build(s)")

	buildTypePtr := buildCmd.String("c", "Release", "build configuration (Debug or Release)")
	sysPtr := buildCmd.String("sys", "MacOS,Linux,Windows", "comma-separated target system (MacOS, Linux, Windows)")
	formatsPtr := buildCmd.String("formats", "Standalone,VST3,AU,LV2,Unity,AAX", "comma-separated build formats (VST3,AU,LV2,Standalone,Unity,AAX)")
	newPtr := buildCmd.Bool("new", false, "perform a clean build (removes build/ and Builds/ folders)")

	buildCmd.Parse(args)

	if !IsFlagValid(buildTypePtr, []string{"Debug", "Release"}) {
		fmt.Println("Error: Invalid build configuration. Must be 'Debug' or 'Release'.")
		buildCmd.Usage()
		os.Exit(1)
	}

	if !IsFlagValid(sysPtr, []string{"MacOS", "Linux", "Windows"}) {
		fmt.Println("Error: Invalid system flag. Must be a comma-separated list of 'MacOS', 'Linux', 'Windows'.")
		buildCmd.Usage()
		os.Exit(1)
	}

	if !IsFlagValid(formatsPtr, []string{"Standalone", "VST3", "AU", "LV2", "Unity", "AAX"}) {
		fmt.Println("Error: Invalid formats flag. Must be a comma-separated list of 'Standalone', 'VST3', 'AU', 'LV2', 'Unity', 'AAX'.")
		buildCmd.Usage()
		os.Exit(1)
	}

	if *batchFilePtr != "" { // batch build logic
		if opPtr == nil || *opPtr == "" {
			fmt.Println("Error: Output path (-OP) is required for batch builds.")
			buildCmd.Usage()
			os.Exit(1)
		}

		if *ppPtr != "" || *jpPtr != "" {
			fmt.Println("Error: Cannot specify both batch file (-b) and single project flags (-PP, -JP).")
			buildCmd.Usage()
			os.Exit(1)
		}

		// TODO: implement batch build logic
		BuildBatch(*batchFilePtr, *opPtr, *newPtr)

		return
	}

	if *ppPtr != "" { // single project build logic
		if *opPtr == "" {
			fmt.Println("Error: Output path (-OP) is required for single project builds.")
			buildCmd.Usage()
			os.Exit(1)
		}

		// TODO: implement single project compilation logic
		BuildPlugin(*ppPtr, *jpPtr, *opPtr, *buildTypePtr, *sysPtr, *formatsPtr, *newPtr)

		return
	}
}

// -------------------------------------------------------------
// HELP MENU
// -------------------------------------------------------------
func printRootHelp() {
	fmt.Println(`MOSAC - JUCE Audio Plugin Build Automator

Usage:
  mosac <command> [options]

Commands:
  create-batch    utility to create a batch file (batch.csv) for multiple builds.
  health          run a diagnostic check on the application state.
  build           execute a build. accepts a CSV batch file (-b) or a single project (-PP, -JP, -OP, -sys, -formats, -new).
  update          updates JUCE framework to the desired version (default: latest).

Use "mosac <command> -h" to see options for a specific command.`)
}

// -------------------------------------------------------------
func IsFlagValid(flagVal *string, allowed []string) bool {
	if flagVal == nil || *flagVal == "" {
		return false
	}
	items := strings.SplitSeq(*flagVal, ",")

	for item := range items {
		cleanItem := strings.TrimSpace(item)
		if cleanItem == "" || !slices.Contains(allowed, cleanItem) {
			return false
		}
	}
	return true
}

// -------------------------------------------------------------
// LOCK SYSTEM
// -------------------------------------------------------------

// attempts to create a lock file atomically. It returns the file descriptor, the path to the lock file, and an error if locked.
func acquireLock() (*os.File, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, "", fmt.Errorf("could not get user home directory: %v", err)
	}

	mosacDir := filepath.Join(homeDir, ".mosac")
	os.MkdirAll(mosacDir, 0755)
	lockPath := filepath.Join(mosacDir, "mosac.lock")

	// os.O_EXCL ensures the file is created ONLY if it doesn't already exist.
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		return nil, lockPath, fmt.Errorf("another instance of MOSAC is already running")
	}

	// write the current PID into the lock file for debugging
	fmt.Fprintf(file, "%d\n", os.Getpid())
	return file, lockPath, nil
}

// closes and removes the lock file
func releaseLock(file *os.File, lockPath string) {
	if file != nil {
		file.Close()
	}
	if lockPath != "" {
		os.Remove(lockPath)
	}
}

// catches termination signals to ensure the lock file is correctly removed before exiting.
func setupSignalHandler(file *os.File, lockPath string) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n[!] Received interrupt signal. Cleaning up lock and exiting...")
		releaseLock(file, lockPath)
		os.Exit(1)
	}()
}

// provvisory functions for build logic, to be implemented in the future
func BuildBatch(batchFilePath string, outputPath string, new bool) {}

func BuildPlugin(projectPath string, jucePath string, outputPath string, buildType string, sys string, formats string, new bool) {
}
