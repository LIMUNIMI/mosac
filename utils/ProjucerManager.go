package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// checks if Projucer is already built in the given JUCE directory. If not, it builds Projucer and returns its path.
func CheckIfProjucerIsAlreadyBuilt(jucePath string) (string, error) {
	projucerPath, err := filepath.Abs(filepath.Join(jucePath, "extras", "Projucer", "Builds", "MacOSX", "build", "Release", "Projucer.app"))
	if err != nil {return "", fmt.Errorf("[MacOSBuilder] Error occurred while resolving Projucer path: %w", err)}

	if _, err := os.Stat(projucerPath); os.IsNotExist(err) {
		fmt.Println("[MacOSBuilder] Projucer not found. Building Projucer...")
		err = buildProjucer(jucePath)
		if err != nil {return "", fmt.Errorf("[MacOSBuilder] Error occurred while building Projucer: %w", err)}
	}

	return projucerPath, nil
}


// builds the Projucer executable from a given JUCE directory.
// it returns an error if the build fails.
func buildProjucer(juceDir string) (error) {
	projucer_xcodeProj, err := filepath.Abs(filepath.Join(juceDir, "extras", "Projucer", "Builds", "MacOSX", "Projucer.xcodeproj"))
	if err != nil {return fmt.Errorf("[Mosac] Error occurred while resolving Projucer Xcode project path: %w", err)}

	builProjucerCmd := []string{"xcodebuild", "-project", projucer_xcodeProj, "-configuration", "Release"}

	cmd := exec.Command(builProjucerCmd[0], builProjucerCmd[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("[Mosac] Building Projucer from JUCE dir: %s\n", juceDir)

	err = cmd.Run()
	if err != nil {return fmt.Errorf("[Mosac] Error occurred while building Projucer: %w", err)}

	fmt.Println("[Mosac] Projucer built successfully.")

	return nil
}


// resaves the Jucer project using --resave
func ResaveProject(projectPath, projucerPath, pluginName string, pluginFormats []string) error {
	jucerFilePath := filepath.Join(projectPath, pluginName+".jucer")

	// substitute the current plugin formats in the Jucer project file with the selected ones (parameter)
	for i, format := range pluginFormats {
		switch format {
		case "Standalone":
			pluginFormats[i] = "buildStandalone"
		case "LV2":
			pluginFormats[i] = "buildLV2"
		case "VST3":
			pluginFormats[i] = "buildVST3"
		case "AU":
			pluginFormats[i] = "buildAU"
		case "Unity":
			pluginFormats[i] = "buildUnity"
		case "AAX":
			pluginFormats[i] = "buildAAX"
		default:
			return fmt.Errorf("[Mosac] Invalid plugin format: %s", format)
		}
	}

	file, err := os.ReadFile(jucerFilePath)
	if err != nil {return fmt.Errorf("[Mosac] Error occurred while reading Jucer project file: %w", err)}
	
	// replace the plugin formats in the Jucer project file
	joined := strings.Join(pluginFormats, ",")
	re := regexp.MustCompile(`pluginFormats\s*=\s*"[^"]*"`)
	replacement := `pluginFormats="` + joined + `"`
	newContent := re.ReplaceAllString(string(file), replacement)
	
	if err = os.WriteFile(jucerFilePath, []byte(newContent), 0o644); err != nil {return fmt.Errorf("[Mosac] Error occurred while writing Jucer project file: %w", err)}

	cmd := exec.Command("."+projucerPath, "--resave", jucerFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("[Mosac] Resaving Jucer project: %s\n", jucerFilePath)
	
	err = cmd.Run()
	if err != nil {return fmt.Errorf("[Mosac] Error occurred while resaving Jucer project: %w", err)}
	return nil
}