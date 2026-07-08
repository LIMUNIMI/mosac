package builders

import (
	"fmt"
	"strings"
	"path/filepath"
	"mosac/utils"
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



func ContainsValidArg(list []string, validArgs map[string]bool) bool {
	seen := make(map[string]bool)
	
	for _, arg := range list {
		arg = strings.TrimSpace(arg)
		if !validArgs[arg] {return false}
		if seen[arg] {return false}
		seen[arg] = true
	}
	
	return len(list) > 0
}



func BuildPlugin(projDir string, juceDir string, outputDir string, buildType string, buildForOS, pluginFormats []string) (error) {
	if projDir == "" || juceDir == "" || outputDir == "" {return fmt.Errorf("[MOSAC] Please provide the required paths for the JUCE project, JUCE directory, and output directory.")}
	if buildType != "Debug" && buildType != "Release" {return fmt.Errorf("[MOSAC] Invalid build type. Please specify either 'Debug' or 'Release'.")}
	if len(buildForOS) == 0 {return fmt.Errorf("[MOSAC] Please specify at least one target OS.")}
	if !ContainsValidArg(buildForOS, validOS) {return fmt.Errorf("[MOSAC] Please specify at least one valid OS (MacOS, Linux, or Windows).")}
	if len(pluginFormats) == 0 {return fmt.Errorf("[MOSAC] Please specify at least one plugin format.")}
	if !ContainsValidArg(pluginFormats, validFormats) {return fmt.Errorf("[MOSAC] Please specify at least one valid plugin format (Standalone, LV2, VST3, AU, Unity, or AAX).")}
	
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



func ProcessBatchFile(batchPath string) error {

	return nil
}