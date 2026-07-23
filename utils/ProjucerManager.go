package utils

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func projucerExecutablePath(projucerAppPath string) string {
	return filepath.Join(projucerAppPath, "Contents", "MacOS", "Projucer")
}

// checks if Projucer is already built in the given JUCE directory. If not, it builds Projucer and returns its path.
func CheckIfProjucerIsAlreadyBuilt(jucePath string) (string, error) {
	projucerPath, err := filepath.Abs(filepath.Join(jucePath, "extras", "Projucer", "Builds", "MacOSX", "build", "Release", "Projucer.app"))
	if err != nil {return "", fmt.Errorf("[MacOSBuilder] Error occurred while resolving Projucer path: %w", err)}

	if _, err := os.Stat(projucerPath); os.IsNotExist(err) {
		fmt.Println("[MacOSBuilder] WARNING!")
		fmt.Println("=================================================\n|| Building Projucer, this may take a while... ||\n=================================================")
		err = buildProjucer(jucePath)
		if err != nil {return "", fmt.Errorf("[MacOSBuilder] Error occurred while building Projucer: %w", err)}
	}

	return projucerExecutablePath(projucerPath), nil
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

// resaves the unique Jucer project in projectPath using --resave
func ResaveProject(projectPath, jucePath, projucerPath string, pluginFormats []string, skipAAX bool) error {
	jucerFilePath, err := getJucerFilePath(projectPath)
	if err != nil {
		return fmt.Errorf("[Mosac] Error occurred while resolving Jucer project file: %w", err)
	}

	err = updateJucerFallbackMetadata(jucerFilePath)
	if err != nil {return fmt.Errorf("[Mosac] Error occurred while normalizing Jucer metadata: %w", err)}

	err = updateJucerModulePaths(jucerFilePath, projectPath, jucePath)
	if err != nil {return fmt.Errorf("[Mosac] Error occurred while updating Jucer module paths: %w", err)}

	projucerFormats := make([]string, len(pluginFormats))
	copy(projucerFormats, pluginFormats)

	// substitute the current plugin formats in the Jucer project file with the selected ones (parameter)
	for i, format := range projucerFormats {
		switch format {
		case "Standalone":
			projucerFormats[i] = "buildStandalone"
		case "LV2":
			projucerFormats[i] = "buildLV2"
		case "VST3":
			projucerFormats[i] = "buildVST3"
		case "AU":
			projucerFormats[i] = "buildAU"
		case "Unity":
			projucerFormats[i] = "buildUnity"
		case "AAX":
			if skipAAX {fmt.Println("[Mosac] AAX format is only supported on MacOS (JUCE8 or newer).");continue}
			projucerFormats[i] = "buildAAX"
		default:
			return fmt.Errorf("[Mosac] Invalid plugin format: %s", format)
		}
	}

	file, err := os.ReadFile(jucerFilePath)
	if err != nil {return fmt.Errorf("[Mosac] Error occurred while reading Jucer project file: %w", err)}
	
	// replace the plugin formats in the Jucer project file
	joined := strings.Join(projucerFormats, ",")
	re := regexp.MustCompile(`pluginFormats\s*=\s*"[^"]*"`)
	replacement := `pluginFormats="` + joined + `"`
	newContent := re.ReplaceAllString(string(file), replacement)
	
	if err = os.WriteFile(jucerFilePath, []byte(newContent), 0o644); err != nil {return fmt.Errorf("[Mosac] Error occurred while writing Jucer project file: %w", err)}

	cmd := exec.Command(projucerPath, "--resave", jucerFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("[Mosac] Resaving Jucer project: %s\n", jucerFilePath)

	err = cmd.Run()
	if err != nil {return fmt.Errorf("[Mosac] Error occurred while resaving Jucer project: %w", err)}
	return nil
}


func updateJucerFallbackMetadata(jucerFilePath string) error {
	file, err := os.Open(jucerFilePath)
	if err != nil {return fmt.Errorf("[Mosac] Error occurred while opening Jucer project file: %w", err)}
	defer file.Close()

	var raw RawJucerProject
	if err := xml.NewDecoder(file).Decode(&raw); err != nil {return fmt.Errorf("[Mosac] Error occurred while parsing Jucer project file: %w", err)}

	normalizeRawJucerProject(&raw, nil)

	content, err := os.ReadFile(jucerFilePath)
	if err != nil {return fmt.Errorf("[Mosac] Error occurred while reading Jucer project file: %w", err)}

	attrs := map[string]string{
		"name":                       getString(raw.Name, "MOSAC"),
		"version":                    getString(raw.Version, "1.0.0"),
		"companyName":                getString(raw.CompanyName, "Laboratorio di Informatica Musicale"),
		"pluginManufacturerCode":     getString(raw.PluginManufacturerCode, "LIM!"),
		"pluginManufacturer":         getString(raw.PluginManufacturer, "LIM"),
		"pluginCode":                 getNormalizedPluginCode(raw),
		"pluginDesc":                 getString(raw.PluginDesc, "insert here italian plugin description"),
		"pluginName":                 getString(raw.PluginName, getString(raw.Name, "MOSAC")),
		"companyEmail":               getString(raw.CompanyEmail, "lim@di.unimi.it"),
		"companyWebsite":             getString(raw.CompanyWebsite, "https://www.lim.di.unimi.it/"),
		"companyCopyright":           getString(raw.CompanyCopyright, getString(raw.CompanyEmail, "lim@di.unimi.it")),
		"binaryDataNamespace":        getString(raw.BinaryDataNamespace, "BinaryData"),
		"includeBinaryInJuceHeader":  strconv.Itoa(getInt(raw.IncludeBinaryInJuceHeader, 1)),
		"pluginVST3Category":         getString(raw.PluginVST3Category, "Fx"),
		"pluginAAXCategory":          getString(raw.PluginAAXCategory, "0"),
		"pluginAUMainType":           getString(raw.PluginAUMainType, "'aufx'"),
		"pluginCharacteristicsValue": getString(raw.PluginCharacteristicsValue, ""),
		"defines":                    getString(raw.Defines, ""),
		"pluginFormats":              getString(raw.PluginFormats, "buildStandalone,buildVST3,buildAU,buildLV2,buildUnity"),
	}

	newContent := string(content)
	for attr, value := range attrs {
		newContent, err = replaceRootXMLAttribute(newContent, attr, value)
		if err != nil {return err}
	}

	if newContent == string(content) {return nil}

	if err = os.WriteFile(jucerFilePath, []byte(newContent), 0o644); err != nil {return fmt.Errorf("[Mosac] Error occurred while writing Jucer project file: %w", err)}

	return nil
}

func replaceRootXMLAttribute(content, attr, value string) (string, error) {
	start := strings.Index(content, "<JUCERPROJECT")
	if start == -1 {return content, nil}

	end := strings.Index(content[start:], ">")
	if end == -1 {return "", fmt.Errorf("[Mosac] malformed Jucer project file: missing closing tag for JUCERPROJECT")}

	openingTag := content[start : start+end+1]
	attrRegex := regexp.MustCompile(attr + `\s*=\s*"[^"]*"`)
	escapedValue := escapeXMLAttribute(value)
	updatedTag := openingTag

	if attrRegex.MatchString(openingTag) {
		updatedTag = attrRegex.ReplaceAllString(openingTag, attr+`="`+escapedValue+`"`)
	} else {
		updatedTag = strings.TrimSuffix(openingTag, ">") + " " + attr + `="` + escapedValue + `">`
	}

	if updatedTag == openingTag {return content, nil}

	return content[:start] + updatedTag + content[start+end+1:], nil
}

func escapeXMLAttribute(value string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(value))
	return buf.String()
}

func getNormalizedPluginCode(raw RawJucerProject) string {
	pluginCode := getString(raw.PluginCode, "Lim0")
	if pluginCode == "Lim0" && raw.ID != nil {pluginCode = getPluginCodeFromUID(*raw.ID)}
	return pluginCode
}

func updateJucerModulePaths(jucerFilePath, projectPath, jucePath string) error {
	file, err := os.ReadFile(jucerFilePath)
	if err != nil {return fmt.Errorf("[Mosac] Error occurred while reading Jucer project file: %w", err)}

	modulesPath, err := filepath.Rel(projectPath, filepath.Join(jucePath, "modules"))
	if err != nil {return fmt.Errorf("[Mosac] Error occurred while resolving JUCE modules path: %w", err)}
	buildToolsPath, err := filepath.Rel(projectPath, filepath.Join(jucePath, "extras", "Build"))
	if err != nil {return fmt.Errorf("[Mosac] Error occurred while resolving JUCE build tools path: %w", err)}

	modulesPath = filepath.ToSlash(modulesPath)
	buildToolsPath = filepath.ToSlash(buildToolsPath)

	lines := strings.Split(string(file), "\n")
	modified := false

	for i, line := range lines {
		if !strings.Contains(line, "<MODULEPATH") || !strings.Contains(line, `id="juce_`) {
			continue
		}

		if strings.Contains(line, `id="juce_build_tools"`) {
			re := regexp.MustCompile(`path="[^"]*"`)
			updated := re.ReplaceAllString(line, `path="`+buildToolsPath+`"`)
			if updated != line {
				lines[i] = updated
				modified = true
			}
			continue
		}

		re := regexp.MustCompile(`path="[^"]*"`)
		updated := re.ReplaceAllString(line, `path="`+modulesPath+`"`)
		if updated != line {
			lines[i] = updated
			modified = true
		}
	}

	if !modified {return nil}

	newContent := strings.Join(lines, "\n")
	if err = os.WriteFile(jucerFilePath, []byte(newContent), 0o644); err != nil {return fmt.Errorf("[Mosac] Error occurred while writing Jucer project file: %w", err)}

	return nil
}
