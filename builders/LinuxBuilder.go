package builders

import (
	"fmt"

	"mosac/utils"

	"github.com/docker/docker/client"
)

type LinuxBuilder struct {
	JucePath    string
	ProjectPath string
	BuildConfiguration string
	
	buildCommand []string

	DockerCli   *client.Client
}



func NewLinuxBuilder(jucePath, projectPath, buildConfiguration string) (*LinuxBuilder, error) {
	cli, err := utils.StartDocker()
	if err != nil {return nil, err}

	return &LinuxBuilder{
		JucePath:     jucePath,
		ProjectPath:  projectPath,
		DockerCli:    cli,

		buildCommand: []string{"sh", "-c",
			fmt.Sprintf(
				"cmake -S /workspace -G Ninja -B build/Linux/%[1]s "+
					"-DCMAKE_BUILD_TYPE=%[1]s "+
					"-DCMAKE_EXE_LINKER_FLAGS=\"-fuse-ld=gold -flto=auto\" "+
					"-DCMAKE_SHARED_LINKER_FLAGS=\"-fuse-ld=gold -flto=auto\" "+
					"-DCMAKE_MODULE_LINKER_FLAGS=\"-fuse-ld=gold -flto=auto\" "+
					"-DJUCE_PATH=/opt -DPROJECT_DIR=/workspace && "+
					"cmake --build /workspace/build/Linux/%[1]s",buildConfiguration,)},
	}, nil
}



func (b *LinuxBuilder) Build() error {
	imageName := "juce-builder:linux"
	targetStage := "go_juce_builder_linux"

	fmt.Println("[LinuxBuilder] Preparing environment...")

	binds := []string{
		fmt.Sprintf("%s:/opt", b.JucePath),
		fmt.Sprintf("%s:/workspace", b.ProjectPath),
	}

	err := utils.RunContainer(b.DockerCli, imageName, targetStage, b.buildCommand, binds)
	if err != nil {return err}

	return nil
}



func (b *LinuxBuilder) Close() {
	if b.DockerCli != nil {b.DockerCli.Close()}
}