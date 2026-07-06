package builders

import (
	"fmt"
	"os"
	"os/exec"
)

type MacOSBuilder struct {	
	buildCommand []string
}

func NewMacOSBuilder(jucePath, projectPath, buildConfiguration string) (*MacOSBuilder, error) {
	return &MacOSBuilder{
		buildCommand: []string{"zsh",	"-c",
			fmt.Sprintf("cmake -S %[1]s -G Xcode -B %[1]s/build/MacOS/%[2]s "+
				"-DCMAKE_OSX_ARCHITECTURES=\"arm64;x86_64\" "+
				"-DCMAKE_OSX_DEPLOYMENT_TARGET=10.13.6 "+
				"-DJUCE_PATH=%[3]s "+
				"-DPROJECT_DIR=%[1]s && "+
				"cmake --build %[1]s/build/MacOS/%[2]s --config %[2]s",
				projectPath,
				buildConfiguration,
				jucePath,)},
	}, nil
}


func (b *MacOSBuilder) Build() error {
	cmd := exec.Command(b.buildCommand[0], b.buildCommand[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("[MacOSBuilder] Starting build process...")

	err := cmd.Run()
	if err != nil {return err}

	fmt.Println("[MacOSBuilder] Build completed succesfully.")

	return nil
}