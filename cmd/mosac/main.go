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

	"mosac/internal/batchMaker"
	"mosac/internal/builders"
	"mosac/internal/health"
	"mosac/internal/juce"
)

func main() {
	os.Exit(run())
}

func run() (exitCode int) {
	// ------- LOCK	------------------------------------------
	lockFile, lockPath, err := acquireLock()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("If you are certain MOSAC is not running, manually delete the file:\n%s\n", lockPath)
		return 1
	}
	defer releaseLock(lockFile, lockPath)
	setupSignalHandler(lockFile, lockPath)
	// --------------------------------------------------------

	if len(os.Args) < 2 {
		printRootHelp()
		return 0
	}

	switch os.Args[1] {
	case "create-batch":
		return createBatchCSVFile(os.Args[2:])
	case "health":
		return runHealthCmd()
	case "build":
		return runBuildCmd(os.Args[2:])
	case "update":
		return runUpdateCmd(os.Args[2:])
	case "-h", "--help", "help":
		printRootHelp()
		return 0
	default:
		fmt.Printf("Error: unknown command '%s'.\n\n", os.Args[1])
		printRootHelp()
		return 1
	}
}

// -------------------------------------------------------------
// SUBCOMMAND: batch
// -------------------------------------------------------------
func createBatchCSVFile(args []string) int {
	batchCmd := flag.NewFlagSet("create-batch", flag.ContinueOnError)

	dirPtr := batchCmd.String("dir", "", "directory containing the projects")
	jucePtr := batchCmd.String("juce", "", "default JUCE version to use")
	outPtr := batchCmd.String("out", "", "output directory for batch.csv file")
	buildTypePtr := batchCmd.String("c", "Release", "build configuration (Debug or Release)")
	sysPtr := batchCmd.String("sys", "MacOS,Linux,Windows", "comma-separated target system")
	formatsPtr := batchCmd.String("formats", "Standalone,VST3,AU,LV2,Unity", "comma-separated build formats")

	if err := batchCmd.Parse(args); err != nil {
		return 1
	}

	if *dirPtr == "" || *jucePtr == "" || *outPtr == "" {
		fmt.Println("Error: Missing required flags for batch command.")
		batchCmd.Usage()
		return 1
	}

	if !IsFlagValid(buildTypePtr, []string{"Debug", "Release"}) {
		fmt.Println("Error: Invalid build configuration. Must be 'Debug' or 'Release'.")
		batchCmd.Usage()
		return 1
	}

	if !IsFlagValid(sysPtr, []string{"MacOS", "Linux", "Windows"}) {
		fmt.Println("Error: Invalid system flag. Must be a comma-separated list of 'MacOS', 'Linux', 'Windows'.")
		batchCmd.Usage()
		return 1
	}

	if !IsFlagValid(formatsPtr, []string{"Standalone", "VST3", "AU", "LV2", "Unity"}) {
		fmt.Println("Error: Invalid formats flag. Must be a comma-separated list of 'Standalone', 'VST3', 'AU', 'LV2', 'Unity', 'AAX'.")
		batchCmd.Usage()
		return 1
	}

	err := batchMaker.CreateBatchCSV(*dirPtr, *jucePtr, *outPtr, *buildTypePtr, *sysPtr, *formatsPtr)
	if err != nil {
		fmt.Printf("Error creating batch file: %v\n", err)
		return 1
	}

	return 0
}

// -------------------------------------------------------------
// SUBCOMMAND: health
// -------------------------------------------------------------
func runHealthCmd() int {
	if err := health.Check(); err != nil {
		fmt.Printf("\n[!] Health check failed: %v\n", err)
		return 1
	}
	return 0
}

// -------------------------------------------------------------
// SUBCOMMAND: update
// -------------------------------------------------------------
func runUpdateCmd(args []string) int {
	updateCmd := flag.NewFlagSet("update", flag.ContinueOnError)
	versionPtr := updateCmd.String("v", "latest", "desired JUCE version to update to (default: latest)")

	if err := updateCmd.Parse(args); err != nil {
		return 1
	}

	fmt.Printf("Updating JUCE to version: %s\n", *versionPtr)
	if err := juce.Update(*versionPtr); err != nil {
		fmt.Printf("Error updating JUCE: %v\n", err)
		return 1
	}
	return 0
}

// -------------------------------------------------------------
// SUBCOMMAND: build
// -------------------------------------------------------------
func runBuildCmd(args []string) int {
	buildCmd := flag.NewFlagSet("build", flag.ContinueOnError)

	batchFilePtr := buildCmd.String("b", "", "path to a batch.csv file")

	ppPtr := buildCmd.String("PP", "", "single project path")
	jpPtr := buildCmd.Int("JV", -1, "JUCE version") // -1 indicates that the version is not specified.
	opPtr := buildCmd.String("OP", "", "output path for the build(s)")

	simpleOutPtr := buildCmd.Bool("simple-out", false, "if specified, only the compiled plugin folder + its json will be copied to the output path. otherwise, the build folder, the json report file and the plugin source folder will be copied to the output path in a 'pluginName' folder containing everything.")

	buildTypePtr := buildCmd.String("c", "Release", "build configuration (Debug or Release)")
	sysPtr := buildCmd.String("sys", "MacOS,Linux,Windows", "comma-separated target system")
	formatsPtr := buildCmd.String("formats", "Standalone,VST3,AU,LV2,Unity", "comma-separated build formats")
	newPtr := buildCmd.Bool("new", false, "perform a clean build (removes build/ and Builds/ folders)")

	if err := buildCmd.Parse(args); err != nil {
		return 1
	}

	if !IsFlagValid(buildTypePtr, []string{"Debug", "Release"}) {
		fmt.Println("Error: Invalid build configuration. Must be 'Debug' or 'Release'.")
		buildCmd.Usage()
		return 1
	}

	if !IsFlagValid(sysPtr, []string{"MacOS", "Linux", "Windows"}) {
		fmt.Println("Error: Invalid system flag. Must be a comma-separated list of 'MacOS', 'Linux', 'Windows'.")
		buildCmd.Usage()
		return 1
	}

	if !IsFlagValid(formatsPtr, []string{"Standalone", "VST3", "AU", "LV2", "Unity", "AAX"}) {
		fmt.Println("Error: Invalid formats flag. Must be a comma-separated list of 'Standalone', 'VST3', 'AU', 'LV2', 'Unity', 'AAX'.")
		buildCmd.Usage()
		return 1
	}

	if *batchFilePtr != "" { // batch build logic
		if opPtr == nil || *opPtr == "" {
			fmt.Println("Error: Output path (-OP) is required for batch builds.")
			buildCmd.Usage()
			return 1
		}

		if *ppPtr != "" || *jpPtr < 1 {
			fmt.Println("Error: Cannot specify both batch file (-b) and single project flags (-PP, -JV).")
			buildCmd.Usage()
			return 1
		}

		if err := builders.BuildBatch(*batchFilePtr, *opPtr, *newPtr, *simpleOutPtr); err != nil {
			fmt.Printf("Error building batch: %v\n", err)
			return 1
		}

		return 0
	}

	if *ppPtr != "" { // single project build logic
		if *opPtr == "" {
			fmt.Println("Error: Output path (-OP) is required for single project builds.")
			buildCmd.Usage()
			return 1
		}

		if err := builders.BuildPlugin(*ppPtr, *jpPtr, *opPtr, *buildTypePtr, strings.Split(*sysPtr, ","), strings.Split(*formatsPtr, ","), *newPtr, *simpleOutPtr); err != nil {
			return 1
		}

		return 0
	}

	return 0
}

// -------------------------------------------------------------
// HELP MENU
// -------------------------------------------------------------
func printRootHelp() {
	fmt.Println(`MOSAC - Multi OS Audio Compiler

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
