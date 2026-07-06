package main

import (
	_ "embed"
	"flag"
	"fmt"
	"mosac/builders"
	"mosac/utils"
	"strings"
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

	projectDir = flag.String("PP", "", "Path to the JUCE project directory")
	juceDir = flag.String("JP", "", "Path to the JUCE directory")
	outputDir = flag.String("OP", "", "Path to the output directory.")
	buildType = flag.String("b", "Release", "Build type [Debug or Release]")
	buildForOS = flag.String("sys", "MacOS,Linux,Windows", "Comma-separated list of targeted OS to build for.")
	pluginFormats = flag.String("formats", "Standalone,LV2,VST3,AU,Unity", "Comma-separated list of plugin formats to include")

	buildImages = flag.Bool("img", false, "use this flag to build required Docker images and stop the program.")
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

	projDir := *projectDir
	juceDir := *juceDir
	outputDir := *outputDir
	buildType := *buildType
	buildForOS := strings.Split(*buildForOS, ",")
	pluginFormats := strings.Split(*pluginFormats, ",")
	
	if projDir == "" || juceDir == "" || outputDir == "" {fmt.Println("[MOSAC] Please provide the required paths for the JUCE project, JUCE directory, and output directory.");return}
	if buildType != "Debug" && buildType != "Release" {fmt.Println("[MOSAC] Invalid build type. Please specify either 'Debug' or 'Release'.");return}
	if len(buildForOS) == 0 {fmt.Println("[MOSAC] Please specify at least one target OS.");return}
	if !ContainsValidArg(buildForOS, validOS) {fmt.Println("[MOSAC] Please specify at least one valid OS (MacOS, Linux, or Windows).");return}
	if len(pluginFormats) == 0 {fmt.Println("[MOSAC] Please specify at least one plugin format.");return}
	if !ContainsValidArg(pluginFormats, validFormats) {fmt.Println("[MOSAC] Please specify at least one valid plugin format (Standalone, LV2, VST3, AU, Unity, or AAX).");return}
	
	// ===== START =====
	fmt.Println("===== MOSAC =====")

	_, err := builders.BuildPlugin(projDir, juceDir, outputDir, buildType, buildForOS, pluginFormats)
	if err != nil {fmt.Printf("[MOSAC] Error occurred while building plugin: %v\n", err)}

}