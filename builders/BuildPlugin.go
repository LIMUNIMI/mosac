package builders

import (
	"fmt"
	"mosac/utils"
)

func BuildPlugin(projDir string, juceDir string, outputDir string, buildType string, buildForOS, pluginFormats []string) (*utils.PluginProject, error) {
	fmt.Printf("Building project at: %s\nSelected JUCE directory: %s\nOutput directory: %s\nBuild type: %s\nTarget OS: %v\nPlugin formats: %v\n\n--- START ---\n", projDir, juceDir, outputDir, buildType, buildForOS, pluginFormats)
	
	// create CMakeLists.txt from Jucer file
	project, err := utils.Jucer2Cmake(projDir, pluginFormats)
	if err != nil {return nil, fmt.Errorf("[BuildPlugin] Error occurred while parsing Jucer file: %w", err)}

	// start build process
	for _, os := range buildForOS {
		switch os {
		case "MacOS":
			builder, err := NewMacOSBuilder(juceDir, projDir, buildType)
			if err != nil {return nil, err}

			err = builder.Build()
			if err != nil {return nil, err}
		case "Linux":
			builder, err := NewLinuxBuilder(juceDir, projDir, buildType)
			if err != nil {return nil, err}
			defer builder.Close()
			
			err = builder.Build()
			if err != nil {return nil, err}
		case "Windows":
			builder, err := NewWindowsBuilder(juceDir, projDir, buildType)
			if err != nil {return nil, err}
			defer builder.Close()

			err = builder.Build()
			if err != nil {return nil, err}
		default:
			return nil, fmt.Errorf("[BuildPlugin] Invalid OS specified: %s", os)
		}
	}

	err = utils.OrganizeOutput(project, projDir, outputDir, buildType, buildForOS, pluginFormats)
	if err != nil {return nil, err}

	fmt.Println("--- END ---")
	return project, nil
}