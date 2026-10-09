package builders

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mosac/internal/output"
	"mosac/internal/utils"
)

var (
	validOS = map[string]bool{
		"MacOS":   true,
		"Linux":   true,
		"Windows": true,
	}

	validFormats = map[string]bool{
		"Standalone": true,
		"LV2":        true,
		"VST3":       true,
		"AU":         true,
		"Unity":      true,
		"AAX":        true,
	}
)

// builds the plugin project for the specified OS and plugin formats
func BuildPlugin(projDir string, juceVer int, outputDir string, buildType string, buildForOS, pluginFormats []string, cleanBuild, simpleOutput bool) error {
	if len(buildForOS) == 0 {
		fmt.Println("[MOSAC] Please specify at least one target OS.")
		return fmt.Errorf("[MOSAC] Please specify at least one target OS.")
	}
	if len(pluginFormats) == 0 {
		fmt.Println("[MOSAC] Please specify at least one plugin format.")
		return fmt.Errorf("[MOSAC] Please specify at least one plugin format.")
	}
	for index, format := range pluginFormats {
		pluginFormats[index] = strings.TrimSpace(format)
	}

	var (
		err             error
		skipLinuxAndWin = false
		buildErrors     []error
	)

	projDir, err = filepath.Abs(projDir)
	if err != nil {
		fmt.Printf("[MOSAC] Error occurred while resolving project directory path: %v", err)
		return err
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		fmt.Printf("[MOSAC] Error occurred while resolving output directory path: %v\n", err)
		return err
	}

	// if juceVer == -1, it means the user did not specify a JUCE version, so it will try to extract it from the .jucer file.
	if juceVer == -1 {
		juceVer = extractJuceVersionFromUserNotes(projDir)
		if juceVer == -1 {
			fmt.Println("[BuildPlugin] No juce version specified and unable to extract from .jucer file. Please specify a JUCE version using the -JV flag.")
			return err
		}
	}

	if juceVer < 1 {
		fmt.Printf("[BuildPlugin] Invalid JUCE version specified: %d. Must be a positive integer.", juceVer)
		return fmt.Errorf("[BuildPlugin] Invalid JUCE version specified: %d. Must be a positive integer.", juceVer)
	}

	// gets the juce directory path based on the specified juce version. If the version is not found, it returns an error.
	juceDir, err := getJucePath(juceVer)
	if err != nil {
		fmt.Printf("[BuildPlugin] Error occurred while resolving JUCE directory path: %v", err)
		return err
	}

	// AAX requires juce version >= 8. juce version is extracted from .jucer file's User Notes section.
	// juce version is overrided by CLI.
	skipAAX := juceVer < 8
	if skipAAX { // removes AAX from pluginFormats if juce version is < 8
		filteredFormats := pluginFormats[:0]
		for _, format := range pluginFormats {
			if format != "AAX" {
				filteredFormats = append(filteredFormats, format)
			}
		}
		pluginFormats = filteredFormats
		if len(pluginFormats) == 0 { // user wants to build only AAX, but JUCE version is < 8
			return fmt.Errorf("[BuildPlugin] AAX requires JUCE 8 or newer")
		}
		// inform the user that AAX will be skipped due to JUCE version
		fmt.Println("[BuildPlugin] AAX format requires JUCE 8 or newer. Skipping AAX.")
	}

	// removes build directories if cleanBuild is true
	if cleanBuild {
		err = os.RemoveAll(filepath.Join(projDir, "build"))
		if err != nil {
			return fmt.Errorf("[BuildPlugin] Error occurred while cleaning build directory: %w", err)
		}

		err = os.RemoveAll(filepath.Join(projDir, "Builds", "MacOSX", "build"))
		if err != nil {
			return fmt.Errorf("[BuildPlugin] Error occurred while cleaning MacOS build directory: %w", err)
		}

		err = os.RemoveAll(filepath.Join(projDir, "JuceLibraryCode"))
		if err != nil {
			return fmt.Errorf("[BuildPlugin] Error occurred while cleaning JuceLibraryCode directory: %w", err)
		}
	}
	cmakeOutputPath := filepath.Join(projDir, "CMakeLists.txt")

	// fix every metadata in the .jucer file, then does --resave (if host is Linux skips ONLY the --resave)
	pluginProject, err := utils.PrepareProject(projDir, juceDir, pluginFormats, skipAAX)
	if err != nil {
		buildErrors = append(buildErrors, fmt.Errorf("[BuildPlugin] Error updating and resaving .jucer file: %w", err))
		goto End
	}

	// using the pluginProject struct metadata, it creates a CMakeLists.txt in cmakeOutputPath.
	err = utils.Jucer2Cmake(cmakeOutputPath, pluginProject)
	if err != nil {
		buildErrors = append(buildErrors, err)
		goto End
	}

	//! START BUILD PROCESS
	fmt.Printf("Building project at: %s\nSelected JUCE directory: %s\nOutput directory: %s\nBuild type: %s\nTarget OS: %v\nPlugin formats: %v\n\n--- START ---\n", projDir, juceDir, outputDir, buildType, buildForOS, pluginFormats)

	// checks if AAX is the only selected format and skips Linux and Windows builds if true
	if len(pluginFormats) == 1 && pluginFormats[0] == "AAX" {
		fmt.Println("[BuildPlugin] AAX format is only supported on MacOS (JUCE8 or newer). Linux and Windows builds will be skipped.")
		skipLinuxAndWin = true
	}

	// start build process
	for _, os := range buildForOS {
		switch os {
		case "MacOS":
			builder, err := NewMacOSBuilder(juceDir, projDir, pluginProject.PluginName, buildType, pluginFormats)
			if err != nil {
				buildErrors = append(buildErrors, fmt.Errorf("[BuildPlugin] Error occurred while creating MacOS builder: %w", err))
				continue
			}

			err = builder.Build()
			if err != nil {
				buildErrors = append(buildErrors, fmt.Errorf("[BuildPlugin] Error occurred while building for MacOS: %w", err))
				continue
			}
		case "Linux":
			if skipLinuxAndWin {
				fmt.Println("[BuildPlugin] Skipping Linux build due to AAX format selection.")
				continue
			}
			builder, err := NewLinuxBuilder(juceDir, projDir, buildType)
			if err != nil {
				buildErrors = append(buildErrors, fmt.Errorf("[BuildPlugin] Error occurred while creating Linux builder: %w", err))
				continue
			}
			defer builder.Close()

			err = builder.Build()
			if err != nil {
				buildErrors = append(buildErrors, fmt.Errorf("[BuildPlugin] Error occurred while building for Linux: %w", err))
				continue
			}
		case "Windows":
			if skipLinuxAndWin {
				fmt.Println("[BuildPlugin] Skipping Windows build due to AAX format selection.")
				continue
			}
			builder, err := NewWindowsBuilder(juceDir, projDir, buildType)
			if err != nil {
				buildErrors = append(buildErrors, fmt.Errorf("[BuildPlugin] Error occurred while creating Windows builder: %w", err))
				continue
			}
			defer builder.Close()

			err = builder.Build()
			if err != nil {
				buildErrors = append(buildErrors, fmt.Errorf("[BuildPlugin] Error occurred while building for Windows: %w", err))
				continue
			}
		default:
			buildErrors = append(buildErrors, fmt.Errorf("[BuildPlugin] Invalid OS specified: %s", os))
		}
	}

	if pluginProject != nil {
		err = output.OrganizeOutput(pluginProject, projDir, outputDir, buildType, buildForOS, pluginFormats, simpleOutput)
		if err != nil {
			return fmt.Errorf("[BuildPlugin] Error occurred while organizing output: %w", err)
		}
	}

End:
	aux := ""

	fmt.Println("--- END ---")
	if len(buildErrors) > 0 {
		fmt.Println("--- ERRORS ---")
		for _, buildErr := range buildErrors {
			fmt.Printf("\t%v\n", buildErr)
			aux += fmt.Sprintf("\t%v\n", buildErr)
		}
		fmt.Println()
		return fmt.Errorf("%s\n", aux)
	}

	return nil
}

// builds multiple plugin projects specified in a batch file
func BuildBatch(batchPath, outputDir string, cleanBuild, simpleOutput bool) error {
	var buildErrors []error
	file, err := os.Open(batchPath)
	if err != nil {
		return fmt.Errorf("[BuildBatch] Error occurred while opening batch file: %w", err)
	}
	defer file.Close()

	var n int
	n = 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		args := strings.Split(line, ",")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if len(args) != 5 {
			fmt.Println("[BuildBatch] Encountered invalid line: ", line)
			break
		}

		projectPath := strings.TrimSpace(args[0])
		juceVerStr := strings.TrimSpace(args[1])
		juceVer, _ := strconv.Atoi(juceVerStr)
		buildType := strings.TrimSpace(args[2])
		buildForOS := strings.Split(strings.TrimSpace(args[3]), ";")
		pluginFormats := strings.Split(strings.TrimSpace(args[4]), ";")

		if len(buildForOS) == 0 {
			buildForOS = append(buildForOS, "MacOS", "Linux", "Windows")
		}
		if len(pluginFormats) == 0 {
			pluginFormats = append(pluginFormats, "Standalone", "LV2", "VST3", "AU", "Unity")
		}

		fmt.Printf("== %d° Plugin ==========\n", n+1)

		compilationErrors := BuildPlugin(projectPath, juceVer, outputDir, buildType, buildForOS, pluginFormats, cleanBuild, simpleOutput)

		// format single n° plugin error
		if compilationErrors != nil {
			buildErrors = append(buildErrors, fmt.Errorf("[Plugin %d] Error occurred while building %s:\n%v", n+1, projectPath[strings.LastIndex(projectPath, "/")+1:], compilationErrors))
		}
		n++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("[BuildBatch] Error occurred while reading batch file: %w", err)
	}

	fmt.Println("===== Finished Batch-Mode =====")

	if len(buildErrors) > 0 {
		fmt.Printf("\n\n--- ERRORS ---\n")
		for _, buildErr := range buildErrors {
			fmt.Printf("%v\n", buildErr)
		}
	}

	return nil
}

// given a project directory, it extracts the JUCE version from the .jucer file's User Notes section (juce:version). If the version is not found, it returns an error.
func extractJuceVersionFromUserNotes(projDir string) int {
	jucerFilePath, err := utils.GetJucerFilePath(projDir)
	if err != nil {
		return -1
	}

	jucerFile, err := os.Open(jucerFilePath)
	if err != nil {
		return -1
	}
	defer jucerFile.Close()

	var raw struct {
		XMLName   xml.Name `xml:"JUCERPROJECT"`
		UserNotes string   `xml:"userNotes,attr"`
	}

	decoder := xml.NewDecoder(jucerFile)
	if err := decoder.Decode(&raw); err != nil {
		return -1
	}

	if raw.UserNotes == "" {
		return -1
	}

	scanner := bufio.NewScanner(strings.NewReader(raw.UserNotes))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if rest, found := strings.CutPrefix(line, "juce:"); found {
			versionStr := strings.TrimSpace(rest)
			version, err := strconv.Atoi(versionStr)
			if err != nil {
				return -1
			}
			return version
		}
	}

	return -1
}

func getJucePath(juceVer int) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("[getJucePath] error finding user home directory: %w", err)
	}

	versionStr := strconv.Itoa(juceVer)
	juceDirPath := filepath.Join(homeDir, ".mosac", "juce", versionStr)
	info, err := os.Stat(juceDirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("[getJucePath] error occurred while resolving JUCE directory path: version %d not found", juceVer)
		}
		return "", fmt.Errorf("[getJucePath] error accessing path %s: %w", juceDirPath, err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("[getJucePath] error: path exists but is not a directory: %s", juceDirPath)
	}

	absPath, err := filepath.Abs(juceDirPath)
	if err != nil {
		return "", fmt.Errorf("[getJucePath] error resolving absolute path: %w", err)
	}

	return absPath, nil
}
