package utils

import (
	"fmt"
	"os/exec"
	"time"
	"context"
	_ "embed"
	"archive/tar"
	"bytes"
	"os"
	"os/user"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

//go:embed Dockerfile
var dockerfile string

// StartDocker checks if Docker is running, if not, attempts to start it on MacOS.
func StartDocker() (*client.Client, error) {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("        [DockerManager] failed to initialize client: %w", err)
	}

	// checks if already running
	_, err = cli.Ping(ctx)
	if err == nil {
		fmt.Println("        [DockerManager] Docker already running.")
		return cli, nil
	}

	// if not running, try to start Docker on host (MacOS)
	cmd := exec.Command("open", "-a", "Docker")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("        [DockerManager] could not run Docker: %w", err)
	}

	fmt.Println("        [DockerManager] Trying to open Docker...")
	for range 30 {
		time.Sleep(1 * time.Second)

		_, err = cli.Ping(ctx)
		if err == nil {
			fmt.Println("        [DockerManager] Docker is running.")
			return cli, nil
		}
	}

	return nil, fmt.Errorf("        [DockerManager] Timeout 30s: docker failed to start")
}

// builds the docker image from the embedded Dockerfile and specified target stage
func BuildImageFromEmbedded(cli *client.Client, imageName string, targetStage string) error {
	ctx := context.Background()

	// check if the image already exists
	filterArgs := filters.NewArgs()
	filterArgs.Add("reference", imageName)

	images, err := cli.ImageList(ctx, image.ListOptions{Filters: filterArgs})
	if err != nil {
		return fmt.Errorf("        [DockerManager] Error while checking existing images: %w", err)
	}
	if len(images) > 0 {
		fmt.Println("        [DockerManager] Image already built.")
		return nil
	}

	// builds the image from the embedded Dockerfile
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	dockerfileBytes := []byte(dockerfile)
	tarHeader := &tar.Header{
		Name: "Dockerfile",
		Size: int64(len(dockerfileBytes)),
	}

	if err := tw.WriteHeader(tarHeader); err != nil {
		return fmt.Errorf("[DockerManager] Error while writing tar header: %w", err)
	}
	if _, err := tw.Write(dockerfileBytes); err != nil {
		return fmt.Errorf("[DockerManager] Error while writing tar file: %w", err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("[DockerManager] Error while closing tar writer: %w", err)
	}

	buildOptions := types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile",
		Remove:     true,
		Target:     targetStage,
	}

	fmt.Printf("        [DockerManager] Starting build for: %s...\n", targetStage)

	res, err := cli.ImageBuild(ctx, buf, buildOptions)
	if err != nil {
		return fmt.Errorf("        [DockerManager] Error while building image: %w", err)
	}
	defer res.Body.Close()

	err = jsonmessage.DisplayJSONMessagesStream(
		res.Body,
		os.Stdout,
		os.Stdout.Fd(),
		false, // isTerminal
		nil,
	)
	if err != nil {
		return fmt.Errorf("[DockerManager] Error while decoding build output: %w", err)
	}

	return err
}

// runs a container from the specified image, executes the provided command, and streams the output to stdout and stderr
func RunContainer(cli *client.Client, imageName, targetStage string, command, binds []string) error {
	ctx := context.Background()

	user, err := user.Current()
	if err != nil {
		return fmt.Errorf("[DockerManager] Could not retrieve host user: %w", err)
	}
	userUIDGID := fmt.Sprintf("%s:%s", user.Uid, user.Gid)

	err = BuildImageFromEmbedded(cli, imageName, targetStage)
	if err != nil {
		return fmt.Errorf("[DockerManager] Couldn't build image: %w", err)
	}

	fmt.Printf("        [DockerManager] Creating container from %s...\n", imageName)

	resp, err := cli.ContainerCreate(
		ctx,
		&container.Config{
			Image: imageName,
			Cmd:   command,
			User:  userUIDGID,
		},
		&container.HostConfig{
			AutoRemove: true,
			Binds:      binds,
		},
		nil, nil, "",
	)
	if err != nil {
		return fmt.Errorf("[DockerManager] Error creating container: %w", err)
	}
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("[DockerManager] Error starting container: %w", err)
	}

	fmt.Printf("        [DockerManager] Container started. Execution of command in progress...\n\n")

	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return fmt.Errorf("[DockerManager] Error opening logs: %w", err)
	}
	defer out.Close()

	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	if err != nil {
		return fmt.Errorf("[DockerManager] Error reading logs: %w", err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("[DockerManager] Error during container waiting: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return fmt.Errorf("BUILD FAILED. Exit code: %d", status.StatusCode)
		}
	}

	fmt.Println("\n        [DockerManager] Build completed successfully. Container has been removed.")
	return nil
}
