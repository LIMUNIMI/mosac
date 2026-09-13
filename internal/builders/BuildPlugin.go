package builders

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mosac/internal/output"
	"mosac/internal/project"
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
func BuildPlugin(projDir string, juceDir string, outputDir string, buildType string, buildForOS, pluginFormats []string, cleanBuild bool) error {
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
		pluginProject   *utils.PluginProject
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

	mosacConf, configFound, err := project.LoadMosacConfig(projDir)
	if err != nil {
		fmt.Printf("[BuildPlugin] Error occurred while reading mosac.conf: %v\n", err)
		return err
	}

	if juceDir != "" { // -JP flag is provided
		juceDir, err = project.ResolveJuceDir(juceDir)
		if err != nil {
			fmt.Printf("[BuildPlugin] Error occurred while resolving JUCE directory path: %v\n", err)
			return fmt.Errorf("[BuildPlugin] Error occurred while resolving JUCE directory path: %w", err)
		}

	} else if configFound { // there is mosac.conf file in the project dir
		juceDir, err = project.ResolveJuceDirFromVersion(mosacConf.JuceVersion)
		if err != nil {
			fmt.Printf("[BuildPlugin] Error occurred while resolving JUCE directory from mosac.conf: %v", err)
			return fmt.Errorf("[BuildPlugin] Error occurred while resolving JUCE directory from mosac.conf: %v\n", err)
		}
	} else {
		fmt.Println("[MOSAC] Please provide the JUCE directory with -JP or add a mosac.conf file in the project directory")
		return fmt.Errorf("[MOSAC] Please provide the JUCE directory with -JP or add a mosac.conf file in the project directory")
	}

	skipAAX := utils.IsJUCEVersionLessThan8(juceDir)
	if skipAAX {
		filteredFormats := pluginFormats[:0]
		for _, format := range pluginFormats {
			if format != "AAX" {
				filteredFormats = append(filteredFormats, format)
			}
		}
		pluginFormats = filteredFormats
		if len(pluginFormats) == 0 {
			return fmt.Errorf("[BuildPlugin] AAX requires JUCE 8 or newer")
		}
		fmt.Println("[BuildPlugin] AAX format requires JUCE 8 or newer. Skipping AAX.")
	}

	fmt.Printf("Building project at: %s\nSelected JUCE directory: %s\nOutput directory: %s\nBuild type: %s\nTarget OS: %v\nPlugin formats: %v\n\n--- START ---\n", projDir, juceDir, outputDir, buildType, buildForOS, pluginFormats)

	if err := utils.PrepareProject(projDir, juceDir, pluginFormats, skipAAX); err != nil {
		buildErrors = append(buildErrors, fmt.Errorf("[BuildPlugin] Error preparing Jucer project: %w", err))
		goto End
	}

	// create CMakeLists.txt from the prepared Jucer file
	pluginProject, err = utils.Jucer2Cmake(projDir, pluginFormats)
	if err != nil {
		buildErrors = append(buildErrors, err)
		goto End
	}

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
		err = output.OrganizeOutput(pluginProject, mosacConf, projDir, outputDir, buildType, buildForOS, pluginFormats)
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
func BuildBatch(batchPath, outputDir string, cleanBuild bool) error {
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
		jucePath := strings.TrimSpace(args[1])
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

		compilationErrors := BuildPlugin(projectPath, jucePath, outputDir, buildType, buildForOS, pluginFormats, cleanBuild)

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
