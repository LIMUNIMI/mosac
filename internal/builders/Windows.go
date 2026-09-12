package builders

import (
	"fmt"
	"os"
	"strings"

	"mosac/internal/utils"

	"github.com/docker/docker/client"
)

type WindowsBuilder struct {
	JucePath           string
	ProjectPath        string
	BuildConfiguration string

	buildCommand []string

	DockerCli *client.Client
}

func NewWindowsBuilder(jucePath, projectPath, buildConfiguration string) (*WindowsBuilder, error) {
	cli, err := utils.StartDocker()
	if err != nil {
		return nil, err
	}

	return &WindowsBuilder{
		JucePath:    jucePath,
		ProjectPath: projectPath,
		DockerCli:   cli,

		buildCommand: []string{"sh", "-c",
			fmt.Sprintf(
				"export LIB=\"/xwin/crt/lib/x86_64;/xwin/sdk/lib/ucrt/x86_64;/xwin/sdk/lib/um/x86_64\" && "+
					"export INCLUDE=\"/xwin/crt/include;/xwin/sdk/include/ucrt;/xwin/sdk/include/shared;/xwin/sdk/include/um;/xwin/sdk/include/winrt\" && "+
					"cmake -S /workspace -G Ninja -B /workspace/build/Windows/%[1]s "+
					"-DCMAKE_BUILD_TYPE=%[1]s "+
					"-DJUCE_PATH=/opt "+
					"-DPROJECT_DIR=/workspace "+
					"-DCMAKE_SYSTEM_NAME=Windows "+
					"-DCMAKE_C_COMPILER=clang-cl "+
					"-DCMAKE_CXX_COMPILER=clang-cl "+
					"-DCMAKE_CROSSCOMPILING=ON "+
					"-DCMAKE_CROSSCOMPILING_EMULATOR=/usr/bin/true "+
					"-DCMAKE_MSVC_RUNTIME_LIBRARY=MultiThreaded "+
					"-DCMAKE_MT=llvm-mt "+
					"-DCMAKE_RC_COMPILER=llvm-rc "+
					"-DCMAKE_C_FLAGS=\"$CL_FLAGS\" "+
					"-DCMAKE_CXX_FLAGS=\"$CL_FLAGS\" && "+
					"cmake --build /workspace/build/Windows/%[1]s -j $(nproc --ignore=1)", buildConfiguration)},
	}, nil
}

// removes AAX label from CMakeLists.txt file to avoid building it. (generates error on Windows cross-compilation)
func removeAAXLabelFromCMakeLists(cmakeFilePath string) error {
	cmakelists, err := os.ReadFile(cmakeFilePath)
	if err != nil {
		return fmt.Errorf("[WindowsBuilder] Error occurred while reading CMakeLists.txt file: %w", err)
	}

	lines := strings.Split(string(cmakelists), "\n")
	modified := false

	// if file contains AAX label, remove it
	for i := range lines {
		if strings.Contains(lines[i], "FORMATS") && strings.Contains(lines[i], "AAX") {
			lines[i] = strings.Replace(lines[i], "AAX", "", 1)
			fmt.Println("        [WindowsBuilder] Removed AAX label from CMakeLists.txt file.")
			modified = true
		}
	}

	// if modified, write the modified content back to the file
	if modified {
		newCmakeLists := strings.Join(lines, "\n")
		err = os.WriteFile(cmakeFilePath, []byte(newCmakeLists), 0644)
		if err != nil {
			return fmt.Errorf("[WindowsBuilder] Error occurred while writing modified CMakeLists.txt file: %w", err)
		}
	}

	return nil
}

func (b *WindowsBuilder) Build() error {
	imageName := "juce-builder:windows"
	targetStage := "go_juce_builder_windows"

	fmt.Println("        [WindowsBuilder] Preparing environment...")

	err := removeAAXLabelFromCMakeLists(fmt.Sprintf("%s/CMakeLists.txt", b.ProjectPath))
	if err != nil {
		return err
	}

	binds := []string{
		fmt.Sprintf("%s:/opt", b.JucePath),
		fmt.Sprintf("%s:/workspace", b.ProjectPath),
	}

	err = utils.RunContainer(b.DockerCli, imageName, targetStage, b.buildCommand, binds)
	if err != nil {
		return err
	}

	return nil
}

func (b *WindowsBuilder) Close() {
	if b.DockerCli != nil {
		b.DockerCli.Close()
	}
}
