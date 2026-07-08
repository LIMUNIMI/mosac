package main

import (
	_ "embed"
	"flag"
	"fmt"
	"mosac/builders"
	"mosac/utils"
	"strings"
	"path/filepath"
)



var (
	projectDir = flag.String("PP", "", "Path to the JUCE project directory")
	juceDir = flag.String("JP", "", "Path to the JUCE directory")
	outputDir = flag.String("OP", "", "Path to the output directory.")
	buildType = flag.String("b", "Release", "Build type [Debug or Release]")
	buildForOS = flag.String("sys", "MacOS;Linux;Windows", "Semicolon-separated list of targeted OS to build for.")
	pluginFormats = flag.String("formats", "Standalone;LV2;VST3;AU;Unity", "Semicolon-separated list of plugin formats to include")

	buildImages = flag.Bool("img", false, "use this flag to build required Docker images and stop the program.")
	batchPath = flag.String("batch", "", "Path to a batch file containing multiple build commands.\nThe structure is:\n\tpathToProject,pathToJuce,buildType,sys,formats\nWhere 'sys' is a semicolon-separated list of OS to build for, and 'formats' is a semicolon-separated list of plugin formats to include.")
)

func main() {
	flag.Parse()

	if *buildImages {
		cli, err := utils.StartDocker()
		if err != nil {fmt.Printf("%v\n", err);return}

		fmt.Println("[MOSAC] Building required Docker images...")
		err = utils.BuildImageFromEmbedded(cli, "juce-builder:linux", "go_juce_builder_linux")
		err = utils.BuildImageFromEmbedded(cli, "juce-builder:windows", "go_juce_builder_windows")
		if err != nil {fmt.Printf("[MOSAC] Error occurred while building Docker images: %v\n", err)}
		return
	}


	// ===== BATCH BUILD =====
	if *batchPath != "" {
		batchPath, err := filepath.Abs(*batchPath)
		if err != nil {fmt.Printf("[MOSAC] Error occurred while resolving batch file path: %v\n", err); return}

		outputDir, err := filepath.Abs(*outputDir)
		if err != nil {fmt.Printf("[MOSAC] Error occurred while resolving output directory path: %v\n", err); return}

		fmt.Printf("===== MOSAC - Batch Mode =====\nBatch file: %s\nOutput directory: %s\n\n--- START ---\n", batchPath, outputDir)

		err = builders.BuildBatch(batchPath, outputDir)
		if err != nil {fmt.Println(err)}
		
		return
	}

	
	// ===== SINGLE BUILD =====
	fmt.Println("===== MOSAC =====")
	
	buildForOS := strings.Split(*buildForOS, ";")
	pluginFormats := strings.Split(*pluginFormats, ";")

	err := builders.BuildPlugin(*projectDir, *juceDir, *outputDir, *buildType, buildForOS, pluginFormats)
	if err != nil {fmt.Println(err)}



}