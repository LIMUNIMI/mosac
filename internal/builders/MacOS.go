package builders

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"mosac/internal/utils"
)

type MacOSBuilder struct {
	buildCommand []string

	projucerPath     string
	xcodeProjectPath string
	pluginFormats    []string
}

func NewMacOSBuilder(jucePath, projectPath, pluginName, buildConfiguration string, pluginFormats []string) (*MacOSBuilder, error) {
	projucerPath, err := utils.CheckIfProjucerIsAlreadyBuilt(jucePath)
	if err != nil {
		return nil, err
	}

	xcodeProjectPath := filepath.Join(projectPath, "Builds", "MacOSX", pluginName+".xcodeproj")

	return &MacOSBuilder{
		projucerPath:     projucerPath,
		xcodeProjectPath: xcodeProjectPath,
		pluginFormats:    pluginFormats,

		buildCommand: []string{"xcodebuild", "-project", xcodeProjectPath, "-configuration", buildConfiguration},
	}, nil
}

func (b *MacOSBuilder) Build() error {
	cmd := exec.Command(b.buildCommand[0], b.buildCommand[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("        [MacOSBuilder] Building plugin using Projucer at: %s\n", b.projucerPath)
	fmt.Printf("        [MacOSBuilder] Building plugin using Xcode project at: %s\n", b.xcodeProjectPath)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("[MacOSBuilder] Error occurred while building plugin: %w", err)
	}

	return nil
}
