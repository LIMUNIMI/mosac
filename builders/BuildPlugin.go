package builders

import (
	"fmt"
	"mosac/utils"
	"os"
	"path/filepath"
	"strings"
	"bufio"
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



func containsValidArg(list []string, validArgs map[string]bool) bool {
	seen := make(map[string]bool)
	
	for _, arg := range list {
		arg = strings.TrimSpace(arg)
		if !validArgs[arg] {return false}
		if seen[arg] {return false}
		seen[arg] = true
	}
	
	return len(list) > 0
}



func printErrors(proj *utils.PluginProject) (error) {
	var (
		err error
		warnings string
	)

	if proj.CompanyEmail == "lim@di.unimi.it" || proj.CompanyEmail == " " || !strings.ContainsAny(proj.CompanyEmail, "@") {warnings += "\t- Company email is empty, invalid or set to default value.\n"}
	if proj.PluginDesc == "insert here italian plugin description" || proj.PluginDesc == " " {warnings += "\t- Plugin description is empty or set to default value.\n"}
	if proj.PluginCode == "Lim0" {warnings += "\t- Plugin code is empty or set to default value.\n"}
	if proj.PluginManufacturerCode == " " || proj.PluginManufacturerCode == "Lim!" {warnings += "\t- Plugin manufacturer code is empty or set to default value.\n"}
	
	if warnings != "" {err = fmt.Errorf("[Warnings] Facultative fields missing:\n%s", warnings)}
	return err
}



func BuildPlugin(projDir string, juceDir string, outputDir string, buildType string, buildForOS, pluginFormats []string, showWarnings, cleanBuild bool) (error) {
	if projDir == "" || juceDir == "" || outputDir == "" {return fmt.Errorf("[MOSAC] Please provide the required paths for the JUCE project, JUCE directory, and output directory.")}
	if buildType != "Debug" && buildType != "Release" {return fmt.Errorf("[MOSAC] Invalid build type. Please specify either 'Debug' or 'Release'.")}
	if len(buildForOS) == 0 {return fmt.Errorf("[MOSAC] Please specify at least one target OS.")}
	if !containsValidArg(buildForOS, validOS) {return fmt.Errorf("[MOSAC] Please specify at least one valid OS (MacOS, Linux, or Windows).")}
	if len(pluginFormats) == 0 {return fmt.Errorf("[MOSAC] Please specify at least one plugin format.")}
	if !containsValidArg(pluginFormats, validFormats) {return fmt.Errorf("[MOSAC] Please specify at least one valid plugin format (Standalone, LV2, VST3, AU, Unity, or AAX).")}
	
	// make every path absolute
	var err error
	projDir, err = filepath.Abs(projDir)
	if err != nil {return fmt.Errorf("[MOSAC] Error occurred while resolving project directory path: %w", err)}
	juceDir, err = filepath.Abs(juceDir)
	if err != nil {return fmt.Errorf("[MOSAC] Error occurred while resolving JUCE directory path: %w", err)}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {return fmt.Errorf("[MOSAC] Error occurred while resolving output directory path: %w", err)}


	fmt.Printf("Building project at: %s\nSelected JUCE directory: %s\nOutput directory: %s\nBuild type: %s\nTarget OS: %v\nPlugin formats: %v\n\n--- START ---\n", projDir, juceDir, outputDir, buildType, buildForOS, pluginFormats)
	
	// create CMakeLists.txt from Jucer file
	project, err := utils.Jucer2Cmake(projDir, pluginFormats)
	if err != nil {return fmt.Errorf("[BuildPlugin] Error occurred while parsing Jucer file: %w", err)}

	if showWarnings {
		err = printErrors(project)
		if err != nil {return err}
	}

	if cleanBuild {
		err = os.RemoveAll(filepath.Join(projDir, "build"))
		if err != nil {return fmt.Errorf("[BuildPlugin] Error occurred while cleaning build directory: %w", err)}
	}

	// start build process
	for _, os := range buildForOS {
		switch os {
		case "MacOS":
			builder, err := NewMacOSBuilder(juceDir, projDir, buildType)
			if err != nil {return fmt.Errorf("[BuildPlugin] Error occurred while creating MacOS builder: %w", err)}

			err = builder.Build()
			if err != nil {return fmt.Errorf("[BuildPlugin] Error occurred while building for MacOS: %w", err)}
		case "Linux":
			builder, err := NewLinuxBuilder(juceDir, projDir, buildType)
			if err != nil {return fmt.Errorf("[BuildPlugin] Error occurred while creating Linux builder: %w", err)}
			defer builder.Close()
			
			err = builder.Build()
			if err != nil {return fmt.Errorf("[BuildPlugin] Error occurred while building for Linux: %w", err)}
		case "Windows":
			builder, err := NewWindowsBuilder(juceDir, projDir, buildType)
			if err != nil {return fmt.Errorf("[BuildPlugin] Error occurred while creating Windows builder: %w", err)}
			defer builder.Close()

			err = builder.Build()
			if err != nil {return fmt.Errorf("[BuildPlugin] Error occurred while building for Windows: %w", err)}
		default:
			return fmt.Errorf("[BuildPlugin] Invalid OS specified: %s", os)
		}
	}

	err = utils.OrganizeOutput(project, projDir, outputDir, buildType, buildForOS, pluginFormats)
	if err != nil {return fmt.Errorf("[BuildPlugin] Error occurred while organizing output: %w", err)}

	fmt.Println("--- END ---")
	return nil
}


// projectPath,jucePath,buildType,buildForOS,pluginFormats
func BuildBatch(batchPath, outputDir string, showWarnings, cleanBuild bool) error {
	var buildErrors []error
	file, err := os.Open(batchPath)
	if err != nil {return fmt.Errorf("[BuildBatch] Error occurred while opening batch file: %w", err)}
	defer file.Close()
	
	var n int
	n=0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		
		line := scanner.Text()
		args := strings.Split(line, ",")
		if len(args) != 5 {return fmt.Errorf("[BuildBatch] Invalid batch file format. See -h or --help for more information.")}

		projectPath := strings.TrimSpace(args[0])
		jucePath := strings.TrimSpace(args[1])
		buildType := strings.TrimSpace(args[2])
		buildForOS := strings.Split(strings.TrimSpace(args[3]), ";")
		pluginFormats := strings.Split(strings.TrimSpace(args[4]), ";")

		if len(buildForOS) == 0 {buildForOS = append(buildForOS, "MacOS", "Linux", "Windows")}
		if len(pluginFormats) == 0 {pluginFormats = append(pluginFormats, "Standalone", "LV2", "VST3", "AU", "Unity")}

		err = BuildPlugin(projectPath, jucePath, outputDir, buildType, buildForOS, pluginFormats, showWarnings, cleanBuild)
		if err != nil {buildErrors = append(buildErrors, fmt.Errorf("[Plugin %d] Error occurred while building %s:\n\t%w", n+1, projectPath[strings.LastIndex(projectPath, "/")+1:], err))}
		n++
	}

	if err := scanner.Err(); err != nil {return fmt.Errorf("[BuildBatch] Error occurred while reading batch file: %w", err)}

	fmt.Println("===== Finished Batch-Mode =====")

	if len(buildErrors) > 0 {
		fmt.Println("[BuildBatch] The following errors occurred during the batch build:")
		for _, buildErr := range buildErrors {
			fmt.Printf("\t%v\n", buildErr)
		}
		return fmt.Errorf("[BuildBatch] Batch finished with errors.")
	}

	return nil
}