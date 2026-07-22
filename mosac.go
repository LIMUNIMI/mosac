package main

import (
	"flag"
	"fmt"
	"mosac/builders"
	"mosac/utils"
	"strings"
	"path/filepath"
)



var (
	projectDir = flag.String("PP", "", "Path to the Plugin Project directory")
	juceDir = flag.String("JP", "", "Path to the JUCE directory")
	outputDir = flag.String("OP", "", "Path to the output directory.")
	buildType = flag.String("b", "Release", "Build type [Debug or Release]")
	buildForOS = flag.String("sys", "MacOS,Linux,Windows", "Comma-separated list of targeted OS to build for.")
	pluginFormats = flag.String("formats", "Standalone,LV2,VST3,AU,Unity,AAX", "Comma-separated list of plugin formats to build.")

	initialize = flag.Bool("initialize", false, "Builds required Docker images and stop the program.")
	batchPath = flag.String("batch", "", "Path to a batch file containing multiple build commands.\nSee README.md for more information on the batch file format.")
	showWarnings = flag.Bool("w", false, "Shows warnings for projects with facultative blank fields [skips the build].")
	cleanBuild = flag.Bool("clean", false, "Cleans the /build directory before building the plugin.")
)

func main() {
	flag.Parse()

	if *initialize {
		cli, err := utils.StartDocker()
		if err != nil {fmt.Printf("%v\n", err);return}

		fmt.Println("[MOSAC] Building required Docker images...")
		err = utils.BuildImageFromEmbedded(cli, "juce-builder:linux", "go_juce_builder_linux")
		err = utils.BuildImageFromEmbedded(cli, "juce-builder:windows", "go_juce_builder_windows")
		if err != nil {fmt.Printf("[MOSAC] Error occurred while building Docker images: %v\n", err)}

		// change JUCE version if updated in the future
		pathJUCE7, err := utils.ResolveJuceDirFromVersion("JUCE7")
		if err != nil {fmt.Printf("[MOSAC] Error occurred while resolving JUCE7 directory: %v\n", err)}
		pathJUCE8, err := utils.ResolveJuceDirFromVersion("JUCE8")
		if err != nil {fmt.Printf("[MOSAC] Error occurred while resolving JUCE8 directory: %v\n", err)}
		_, err = utils.CheckIfProjucerIsAlreadyBuilt(pathJUCE7)
		if err != nil {fmt.Printf("[MOSAC] Error occurred while checking Projucer for JUCE7: %v\n", err)}
		_, err = utils.CheckIfProjucerIsAlreadyBuilt(pathJUCE8)
		if err != nil {fmt.Printf("[MOSAC] Error occurred while checking Projucer for JUCE8: %v\n", err)}
		return
	}


	// ===== BATCH BUILD =====
	if *batchPath != "" {
		batchPath, err := filepath.Abs(*batchPath)
		if err != nil {fmt.Printf("[MOSAC] Error occurred while resolving batch file path: %v\n", err); return}

		outputDir, err := filepath.Abs(*outputDir)
		if err != nil {fmt.Printf("[MOSAC] Error occurred while resolving output directory path: %v\n", err); return}

		fmt.Printf("===== MOSAC - Batch Mode =====\nBatch file: %s\nOutput directory: %s\n==============================\n\n", batchPath, outputDir)

		err = builders.BuildBatch(batchPath, outputDir, *showWarnings, *cleanBuild)
		if err != nil {fmt.Println(err)}
		
		return
	}

	// ===== SINGLE BUILD =====
	fmt.Println("===== MOSAC =====")

	buildForOS := strings.Split(*buildForOS, ",")
	pluginFormats := strings.Split(*pluginFormats, ",")

	err := builders.BuildPlugin(*projectDir, *juceDir, *outputDir, *buildType, buildForOS, pluginFormats, *showWarnings, *cleanBuild)
	if err != nil {fmt.Println(err)}
}