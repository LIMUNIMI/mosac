package utils

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

// fixes every metadata in the .jucer file, then does --resave (if host is Linux skips ONLY the --resave)
func PrepareProject(projectPath, jucePath string, pluginFormats []string, skipAAX bool) (*PluginProject, error) {
	jucerFilePath, err := GetJucerFilePath(projectPath)
	if err != nil {
		return nil, fmt.Errorf("[PrepareProject] error occurred while resolving .jucer project file: %w", err)
	}

	if err := updateJucerFallbackMetadata(jucerFilePath); err != nil {
		return nil, fmt.Errorf("[PrepareProject] error occurred while normalizing .jucer metadata: %w", err)
	}
	if err := updateJucerModuleUseGlobalPath(jucerFilePath); err != nil {
		return nil, fmt.Errorf("[PrepareProject] error occurred while updating .jucer module usage: %w", err)
	}
	if err := updateJucerModulePaths(jucerFilePath, projectPath, jucePath); err != nil {
		return nil, fmt.Errorf("[PrepareProject] error occurred while updating .jucer module paths: %w", err)
	}
	if err := updateJucerPluginFormats(jucerFilePath, pluginFormats, skipAAX); err != nil {
		return nil, fmt.Errorf("[PrepareProject] error occurred while updating .jucer plugin formats: %w", err)
	}

	pluginProject, err := parseJucerFile(jucerFilePath)
	if err != nil {
		return nil, fmt.Errorf("[PrepareProject] error while parsing .jucer file: %w", err)
	}

	if runtime.GOOS != "darwin" {
		return pluginProject, nil
	}

	projucerPath, err := CheckIfProjucerIsAlreadyBuilt(jucePath) //TODO ok
	if err != nil {
		return nil, err
	}

	return pluginProject, resaveProject(jucerFilePath, projucerPath)
}

func projucerExecutablePath(projucerAppPath string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(projucerAppPath, "Contents", "MacOS", "Projucer")
	}
	return filepath.Join(filepath.Dir(projucerAppPath), "Projucer")
}

func IsJUCEVersionLessThan8(jucePath string) bool {
	re := regexp.MustCompile(`JUCE-?([0-9]+)`)
	matches := re.FindStringSubmatch(jucePath)
	if len(matches) != 2 {
		return false
	}

	version, err := strconv.Atoi(matches[1])
	return err == nil && version < 8
}

// checks if Projucer is built in the given JUCE directory. If not, it builds Projucer and returns its path.
func CheckIfProjucerIsAlreadyBuilt(jucePath string) (string, error) {
	projucerAppPath, err := filepath.Abs(filepath.Join(jucePath, "extras", "Projucer", "Builds", "MacOSX", "build", "Release", "Projucer.app"))
	if err != nil {
		return "", fmt.Errorf("[CheckIfProjucerIsAlreadyBuilt] error occurred while resolving Projucer path: %w", err)
	}

	if _, err := os.Stat(projucerAppPath); os.IsNotExist(err) {
		if runtime.GOOS != "darwin" {
			return "", fmt.Errorf("[CheckIfProjucerIsAlreadyBuilt] cannot build Projucer for macOS on a non-macOS host (%s)", runtime.GOOS)
		}

		fmt.Println("[CheckIfProjucerIsAlreadyBuilt] WARNING!")
		fmt.Println("=================================================")
		fmt.Println("|| Building Projucer, this may take a while... ||")
		fmt.Println("=================================================")

		err = buildProjucer(jucePath)
		if err != nil {
			return "", fmt.Errorf("[CheckIfProjucerIsAlreadyBuilt] error occurred while building Projucer: %w", err)
		}
	}

	return projucerExecutablePath(projucerAppPath), nil
}

// builds the Projucer executable from a given JUCE directory using xcodebuild. It returns an error if the build fails.
func buildProjucer(juceDir string) error {
	projucerXcodeProj, err := filepath.Abs(filepath.Join(juceDir, "extras", "Projucer", "Builds", "MacOSX", "Projucer.xcodeproj"))
	if err != nil {
		return fmt.Errorf("[buildProjucer] error occurred while resolving Projucer Xcode project path: %w", err)
	}

	buildProjucerCmd := []string{"xcodebuild", "-project", projucerXcodeProj, "-configuration", "Release", "MACOSX_DEPLOYMENT_TARGET=12.0"}

	cmd := exec.Command(buildProjucerCmd[0], buildProjucerCmd[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("[buildProjucer] Building Projucer from JUCE dir: %s\n", juceDir)

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("[buildProjucer] error occurred while building Projucer: %w", err)
	}

	fmt.Println("[buildProjucer] Projucer built successfully.")
	return nil
}

// resaves the unique Jucer project in projectPath using the Projucer --resave command.
func resaveProject(jucerFilePath, projucerPath string) error {
	cmd := exec.Command(projucerPath, "--resave", jucerFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("[resaveProject] Resaving Jucer project: %s\n", jucerFilePath)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("[resaveProject] error occurred while resaving Jucer project: %w", err)
	}

	return nil
}

// changes the plugin formats in the .jucer file with the given array of pluginFormats (from CLI)
func updateJucerPluginFormats(jucerFilePath string, pluginFormats []string, skipAAX bool) error {
	projucerFormats := make([]string, 0, len(pluginFormats))
	for _, format := range pluginFormats {
		format = strings.TrimSpace(format)
		switch format {
		case "Standalone":
			projucerFormats = append(projucerFormats, "buildStandalone")
		case "LV2":
			projucerFormats = append(projucerFormats, "buildLV2")
		case "VST3":
			projucerFormats = append(projucerFormats, "buildVST3")
		case "AU":
			projucerFormats = append(projucerFormats, "buildAU")
		case "Unity":
			projucerFormats = append(projucerFormats, "buildUnity")
		case "AAX":
			if skipAAX {
				continue
			}
			projucerFormats = append(projucerFormats, "buildAAX")
		default:
			return fmt.Errorf("[updateJucerPluginFormats] invalid plugin format: %s", format)
		}
	}

	content, err := os.ReadFile(jucerFilePath)
	if err != nil {
		return fmt.Errorf("[updateJucerPluginFormats] error reading Jucer project file: %w", err)
	}

	updatedContent, err := replaceRootXMLAttribute(
		string(content),
		"pluginFormats",
		strings.Join(projucerFormats, ","),
	)
	if err != nil {
		return err
	}
	if updatedContent == string(content) {
		return nil
	}

	if err := os.WriteFile(jucerFilePath, []byte(updatedContent), 0o644); err != nil {
		return fmt.Errorf("[updateJucerPluginFormats] error writing Jucer project file: %w", err)
	}
	return nil
}

func updateJucerFallbackMetadata(jucerFilePath string) error {
	file, err := os.Open(jucerFilePath)
	if err != nil {
		return fmt.Errorf("[updateJucerFallbackMetadata] error occurred while opening .jucer project file: %w", err)
	}
	defer file.Close()

	var raw RawJucerProject
	if err := xml.NewDecoder(file).Decode(&raw); err != nil {
		return fmt.Errorf("[updateJucerFallbackMetadata] error occurred while parsing Jucer project file: %w", err)
	}

	content, err := os.ReadFile(jucerFilePath)
	if err != nil {
		return fmt.Errorf("[updateJucerFallbackMetadata] error occurred while reading .jucer project file: %w", err)
	}

	attrs := map[string]string{
		"name":                       getString(raw.Name, "BasicAudioPlugin"),
		"version":                    getString(raw.Version, "1.0.0"),
		"companyName":                getString(raw.CompanyName, "LIM"),
		"pluginManufacturerCode":     getString(raw.PluginManufacturerCode, "LIM!"),
		"pluginManufacturer":         getString(raw.PluginManufacturer, "LIM"),
		"pluginCode":                 getNormalizedPluginCode(raw),
		"pluginDesc":                 getString(raw.PluginDesc, "No description available."),
		"pluginName":                 getString(raw.PluginName, getString(raw.Name, "BasicAudioPlugin")),
		"companyEmail":               getString(raw.CompanyEmail, "lim@di.unimi.it"),
		"companyWebsite":             getString(raw.CompanyWebsite, "https://audioplugins.lim.di.unimi.it/"),
		"companyCopyright":           getString(raw.CompanyCopyright, getString(raw.CompanyName, "LIM")),
		"binaryDataNamespace":        getString(raw.BinaryDataNamespace, "BinaryData"),
		"includeBinaryInJuceHeader":  strconv.Itoa(getInt(raw.IncludeBinaryInJuceHeader, 1)),
		"pluginVST3Category":         getString(raw.PluginVST3Category, "Fx"),
		"pluginAAXCategory":          getString(raw.PluginAAXCategory, "0"),
		"pluginAUMainType":           getString(raw.PluginAUMainType, "'aufx'"),
		"pluginCharacteristicsValue": getString(raw.PluginCharacteristicsValue, ""),
		"defines":                    getString(raw.Defines, ""),
		"pluginFormats":              getString(raw.PluginFormats, "buildStandalone,buildVST3,buildAU,buildLV2,buildUnity,buildAAX"),
	}

	// correct spaces to dashes in some metadata fields
	attrs["companyName"] = strings.ReplaceAll(attrs["companyName"], " ", "-")
	attrs["name"] = strings.ReplaceAll(attrs["name"], " ", "-")
	attrs["pluginName"] = strings.ReplaceAll(attrs["pluginName"], " ", "-")
	if !strings.HasPrefix(attrs["companyWebsite"], "https://") {
		attrs["companyWebsite"] = "https://" + attrs["companyWebsite"]
	}

	newContent := string(content)
	for attr, value := range attrs {
		newContent, err = replaceRootXMLAttribute(newContent, attr, value)
		if err != nil {
			return err
		}
	}

	if newContent == string(content) {
		return nil
	}

	if err = os.WriteFile(jucerFilePath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("[updateJucerFallbackMetadata] error occurred while writing Jucer project file: %w", err)
	}

	return nil
}

func replaceRootXMLAttribute(content, attr, value string) (string, error) {
	start := strings.Index(content, "<JUCERPROJECT")
	if start == -1 {
		return content, nil
	}

	end := strings.Index(content[start:], ">")
	if end == -1 {
		return "", fmt.Errorf("[replaceRootXMLAttribute] malformed Jucer project file: missing closing tag for JUCERPROJECT")
	}

	openingTag := content[start : start+end+1]
	attrRegex := regexp.MustCompile(attr + `\s*=\s*"[^"]*"`)
	escapedValue := escapeXMLAttribute(value)
	updatedTag := openingTag

	if attrRegex.MatchString(openingTag) {
		updatedTag = attrRegex.ReplaceAllString(openingTag, attr+`="`+escapedValue+`"`)
	} else {
		updatedTag = strings.TrimSuffix(openingTag, ">") + " " + attr + `="` + escapedValue + `">`
	}

	if updatedTag == openingTag {
		return content, nil
	}

	return content[:start] + updatedTag + content[start+end+1:], nil
}

func escapeXMLAttribute(value string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(value))
	return buf.String()
}

func getNormalizedPluginCode(raw RawJucerProject) string {
	pluginCode := getString(raw.PluginCode, "Lim0")
	if pluginCode == "Lim0" && raw.ID != nil {
		pluginCode = getPluginCodeFromUID(*raw.ID) // Assumed to be in PluginXml.go
	}
	return pluginCode
}

// changes the useGlobalPath attribute of all <MODULE> elements to "0".
func updateJucerModuleUseGlobalPath(jucerFilePath string) error {
	content, err := os.ReadFile(jucerFilePath)
	if err != nil {
		return fmt.Errorf("[updateJucerModuleUseGlobalPath] error occurred while reading Jucer project file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	modified := false

	for i, line := range lines {
		if strings.Contains(line, `useGlobalPath="1"`) {
			modified = true
			lines[i] = strings.ReplaceAll(line, `useGlobalPath="1"`, `useGlobalPath="0"`)
		}
	}

	if !modified {
		return nil
	}

	newContent := strings.Join(lines, "\n")
	if err = os.WriteFile(jucerFilePath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("[updateJucerModuleUseGlobalPath] error occurred while writing Jucer project file: %w", err)
	}

	return nil
}

// updates the paths of all <MODULEPATH> elements to point to the specified jucePath modules.
func updateJucerModulePaths(jucerFilePath, projectPath, jucePath string) error {
	file, err := os.ReadFile(jucerFilePath)
	if err != nil {
		return fmt.Errorf("[updateJucerModulePaths] error occurred while reading Jucer project file: %w", err)
	}

	modulesPath, err := filepath.Rel(projectPath, filepath.Join(jucePath, "modules"))
	if err != nil {
		return fmt.Errorf("[updateJucerModulePaths] error occurred while resolving JUCE modules path: %w", err)
	}

	buildToolsPath, err := filepath.Rel(projectPath, filepath.Join(jucePath, "extras", "Build"))
	if err != nil {
		return fmt.Errorf("[updateJucerModulePaths] error occurred while resolving JUCE build tools path: %w", err)
	}

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

	if !modified {
		return nil
	}

	newContent := strings.Join(lines, "\n")
	if err = os.WriteFile(jucerFilePath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("[updateJucerModulePaths] error occurred while writing Jucer project file: %w", err)
	}

	return nil
}

// parses the .jucer file and returns a PluginProject struct with the parsed data.
func parseJucerFile(jucerFilePath string) (proj *PluginProject, err error) {
	var raw RawJucerProject

	file, err := os.Open(jucerFilePath)
	if err != nil {
		return nil, fmt.Errorf("[parseJucerFile] error while opening file %s: %w", jucerFilePath, err)
	}
	defer file.Close()

	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("[parseJucerFile] error while parsing XML: %w", err)
	}

	proj = &PluginProject{}

	proj.ID = getString(raw.ID, "")
	proj.Name = getString(raw.Name, "")
	proj.Version = getString(raw.Version, "1.0.0")

	proj.PluginName = getString(raw.PluginName, proj.Name)
	proj.PluginManufacturer = getString(raw.PluginManufacturer, "LIM")
	proj.PluginManufacturerCode = getString(raw.PluginManufacturerCode, "LIM!")
	proj.PluginCode = getNormalizedPluginCode(raw)
	proj.PluginDesc = getString(raw.PluginDesc, "No description available.")
	proj.UserNotes = getString(raw.UserNotes, "owner:\nrating:\n")

	proj.Owner_json, proj.Rating_json = parseUserNotes(proj.UserNotes)

	proj.CompanyName = getString(raw.CompanyName, "")
	proj.CompanyEmail = getString(raw.CompanyEmail, "")
	proj.CompanyWebsite = getString(raw.CompanyWebsite, "")

	if copyright := getString(raw.CompanyCopyright, ""); copyright != "" {
		for _, c := range strings.Split(copyright, ",") {
			proj.CompanyCopyright = append(proj.CompanyCopyright, strings.TrimSpace(c))
		}
	}

	// parse pluginFormats (AAX can't be put in CMakeLists for lack of support on Linux and Win)
	formats := strings.Split(*raw.PluginFormats, ",")
	formatMapping := map[string]string{
		"buildVST3":       "VST3",
		"buildAU":         "AU",
		"buildStandalone": "Standalone",
		"buildLV2":        "LV2",
		"buildUnity":      "Unity",
	}
	for _, f := range formats {
		if mapped, ok := formatMapping[f]; ok {
			proj.PluginFormats = append(proj.PluginFormats, mapped)
		}
	}

	// plugin characteristics
	charStr := getString(raw.PluginCharacteristicsValue, "")
	characteristics := strings.Split(charStr, ",")

	checkChar := func(key string) string {
		if slices.Contains(characteristics, key) {
			return "TRUE"
		}
		return "FALSE"
	}

	proj.EditorRequiresKeys = checkChar("pluginEditorRequiresKeys")
	proj.IsMidiEffect = checkChar("pluginIsMidiEffectPlugin")
	proj.IsSynth = checkChar("pluginIsSynth")
	proj.WantsMidiInput = checkChar("pluginWantsMidiIn")
	proj.ProducesMidiOut = checkChar("pluginProducesMidiOut")

	// defaults
	proj.BinaryDataNamespace = getString(raw.BinaryDataNamespace, "BinaryData")
	proj.IncludeBinaryInJuceHeader = getInt(raw.IncludeBinaryInJuceHeader, 1)

	// VST3 category
	proj.PluginVST3Category = strings.Split(getString(raw.PluginVST3Category, "Fx"), ",")

	//! FOR FUTURE SUPPORT:  AAX category
	aaxCategoriesRaw := strings.Split(getString(raw.PluginAAXCategory, "0"), ",")
	var parsedAax []string
	for _, c := range aaxCategoriesRaw {
		switch c {
		case "0":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_None")
		case "1":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_EQ")
		case "2":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_Dynamics")
		case "4":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_PitchShift")
		case "8":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_Reverb")
		case "16":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_Delay")
		case "32":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_Modulation")
		case "64":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_Harmonic")
		case "128":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_NoiseReduction")
		case "256":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_Dither")
		case "512":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_SoundField")
		case "1024":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_HWGenerators")
		case "2048":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_SWGenerators")
		case "4096":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_WrappedPlugin")
		case "8192":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_Effect")
		case "65536":
			parsedAax = append(parsedAax, "AAX_ePlugInCategory_MIDIEffect")
		}
	}
	proj.PluginAAXCategory = parsedAax

	// AU main type
	auTypeStr := getString(raw.PluginAUMainType, "'aufx'")
	switch auTypeStr {
	case "'aufx'":
		proj.PluginAUMainType = "kAudioUnitType_Effect"
	case "'aufc'":
		proj.PluginAUMainType = "kAudioUnitType_FormatConverter"
	case "'augn'":
		proj.PluginAUMainType = "kAudioUnitType_Generator"
	case "'aumi'":
		proj.PluginAUMainType = "kAudioUnitType_MIDIProcessor"
	case "'aumx'":
		proj.PluginAUMainType = "kAudioUnitType_Mixer"
	case "'aumu'":
		proj.PluginAUMainType = "kAudioUnitType_MusicDevice"
	case "'aumf'":
		proj.PluginAUMainType = "kAudioUnitType_MusicEffect"
	case "'auou'":
		proj.PluginAUMainType = "kAudioUnitType_Output"
	case "'aupn'":
		proj.PluginAUMainType = "kAudioUnitType_Panner"
	default:
		proj.PluginAUMainType = "'aufx'"
	}

	// defines
	definesStr := getString(raw.Defines, "")
	if definesStr != "" {
		rawDefines := strings.Fields(definesStr)
		for _, d := range rawDefines {
			if d != "=" && d != "1" {
				cleanedStr := strings.ReplaceAll(d, "=1", "")
				proj.Defines = append(proj.Defines, cleanedStr)
			}
		}
	}

	// modules
	for _, mod := range raw.Modules {
		if mod.ID != "" {
			proj.Modules = append(proj.Modules, "juce::"+mod.ID)
		}
	}

	// external libraries and asset files
	var walkGroup func(g RawGroup)
	walkGroup = func(g RawGroup) {
		for _, f := range g.Files {
			if f.File != "" && f.Resource == "1" { // asset
				proj.AssetFiles = append(proj.AssetFiles, f.File)
			} else if f.File != "" && strings.Contains(f.File, "Libraries") { // external libraries
				proj.LibrarySources = append(proj.LibrarySources, f.File)
			}
		}
		for _, childGroup := range g.Groups {
			walkGroup(childGroup)
		}
	}
	walkGroup(raw.MainGroup)

	return proj, nil
}

// returns the absolute path of the .jucer file in the specified project directory, or an error if not found or if multiple .jucer files are present.
func GetJucerFilePath(ProjectPath string) (string, error) {
	files, err := os.ReadDir(ProjectPath)
	if err != nil {
		return "", fmt.Errorf("[getJucerFilePath] Error while opening dir path %s: %w", ProjectPath, err)
	}

	var jucerFiles []string

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".jucer") {
			jucerFiles = append(jucerFiles, file.Name())
		}
	}

	count := len(jucerFiles)
	if count == 0 {
		return "", fmt.Errorf("[getJucerFilePath] No .jucer file found in directory: %s", ProjectPath)
	}
	if count > 1 {
		return "", fmt.Errorf("[getJucerFilePath] Too many .jucer files (%d) found in directory: %s", count, ProjectPath)
	}

	absPath, err := filepath.Abs(filepath.Join(ProjectPath, jucerFiles[0]))
	if err != nil {
		return "", fmt.Errorf("[getJucerFilePath] Error while resolving absolute path: %w", err)
	}

	return absPath, nil
}
