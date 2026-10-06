package utils

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func RemoveAAXLabelFromCMakeLists(cmakeFilePath string) error {
	cmakeLists, err := os.ReadFile(cmakeFilePath)
	if err != nil {
		return fmt.Errorf("error reading CMakeLists.txt: %w", err)
	}

	lines := strings.Split(string(cmakeLists), "\n")
	modified := false
	for i := range lines {
		if strings.Contains(lines[i], "FORMATS") && strings.Contains(lines[i], "AAX") {
			lines[i] = strings.Replace(lines[i], "AAX", "", 1)
			modified = true
		}
	}

	if !modified {
		return nil
	}

	if err := os.WriteFile(cmakeFilePath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return fmt.Errorf("error writing CMakeLists.txt: %w", err)
	}
	return nil
}

// returns the absolute path of the .jucer file in the specified project directory, or an error if not found or if multiple .jucer files are present.
func getJucerFilePath(ProjectPath string) (string, error) {
	files, err := os.ReadDir(ProjectPath)
	if err != nil {
		return "", fmt.Errorf("[Jucer2Cmake] Error while opening dir path %s: %w", ProjectPath, err)
	}

	var jucerFiles []string

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".jucer") {
			jucerFiles = append(jucerFiles, file.Name())
		}
	}

	count := len(jucerFiles)
	if count == 0 {
		return "", fmt.Errorf("[Jucer2Cmake] No .jucer file found in directory: %s", ProjectPath)
	}
	if count > 1 {
		return "", fmt.Errorf("[Jucer2Cmake] Too many .jucer files (%d) found in directory: %s", count, ProjectPath)
	}

	absPath, err := filepath.Abs(filepath.Join(ProjectPath, jucerFiles[0]))
	if err != nil {
		return "", fmt.Errorf("[Jucer2Cmake] Error while calculating absolute path: %w", err)
	}

	return absPath, nil
}

func normalizeRawJucerProject(raw *RawJucerProject, pluginFormats []string) {
	if raw.Name == nil {
		defaultName := "MOSAC"
		raw.Name = &defaultName
	}
	if raw.Version == nil {
		defaultVersion := "1.0.0"
		raw.Version = &defaultVersion
	}
	if raw.CompanyName == nil {
		defaultCompanyName := "LIM"
		raw.CompanyName = &defaultCompanyName
	}
	if raw.PluginManufacturerCode == nil {
		defaultPluginManufacturerCode := "LIM!"
		raw.PluginManufacturerCode = &defaultPluginManufacturerCode
	}
	if raw.PluginManufacturer == nil {
		defaultPluginManufacturer := "LIM"
		raw.PluginManufacturer = &defaultPluginManufacturer
	}
	if raw.PluginCode == nil {
		defaultPluginCode := "Lim0"
		if raw.ID != nil {
			defaultPluginCode = getPluginCodeFromUID(*raw.ID)
		}
		raw.PluginCode = &defaultPluginCode
	}
	if raw.PluginDesc == nil {
		defaultPluginDesc := "insert here italian plugin description"
		raw.PluginDesc = &defaultPluginDesc
	}
	if raw.PluginName == nil {
		defaultPluginName := getString(raw.Name, "MOSAC")
		raw.PluginName = &defaultPluginName
	}
	if raw.CompanyEmail == nil {
		defaultCompanyEmail := "lim@di.unimi.it"
		raw.CompanyEmail = &defaultCompanyEmail
	}
	if raw.CompanyWebsite == nil {
		defaultCompanyWebsite := "https://www.lim.di.unimi.it/"
		raw.CompanyWebsite = &defaultCompanyWebsite
	} else {
		companyWebsite := strings.TrimSpace(*raw.CompanyWebsite)
		if companyWebsite != "" && !strings.HasPrefix(companyWebsite, "https://") {
			companyWebsite = strings.TrimPrefix(companyWebsite, "http://")
			companyWebsite = strings.TrimPrefix(companyWebsite, "https://")
			companyWebsite = "https://" + companyWebsite
		}
		raw.CompanyWebsite = &companyWebsite
	}
	if raw.CompanyCopyright == nil {
		defaultCompanyCopyright := getString(raw.CompanyEmail, "lim@di.unimi.it")
		raw.CompanyCopyright = &defaultCompanyCopyright
	}
	if raw.BinaryDataNamespace == nil {
		defaultBinaryDataNamespace := "BinaryData"
		raw.BinaryDataNamespace = &defaultBinaryDataNamespace
	}
	if raw.IncludeBinaryInJuceHeader == nil {
		defaultIncludeBinaryInJuceHeader := 1
		raw.IncludeBinaryInJuceHeader = &defaultIncludeBinaryInJuceHeader
	}
	if raw.PluginVST3Category == nil {
		defaultPluginVST3Category := "Fx"
		raw.PluginVST3Category = &defaultPluginVST3Category
	}
	if raw.PluginAAXCategory == nil {
		defaultPluginAAXCategory := "0"
		raw.PluginAAXCategory = &defaultPluginAAXCategory
	}
	if raw.PluginAUMainType == nil {
		defaultPluginAUMainType := "'aufx'"
		raw.PluginAUMainType = &defaultPluginAUMainType
	}
	if raw.PluginCharacteristicsValue == nil {
		defaultPluginCharacteristicsValue := ""
		raw.PluginCharacteristicsValue = &defaultPluginCharacteristicsValue
	}
	if raw.Defines == nil {
		defaultDefines := ""
		raw.Defines = &defaultDefines
	}
	if raw.PluginFormats == nil && len(pluginFormats) == 0 {
		defaultPluginFormats := "buildStandalone,buildVST3,buildAU,buildLV2,buildUnity"
		raw.PluginFormats = &defaultPluginFormats
	}
}

func parseJucerFile(jucerFilePath string, pluginFormats []string) (proj *PluginProject, err error) {
	var raw RawJucerProject

	file, err := os.Open(jucerFilePath)
	if err != nil {
		return nil, fmt.Errorf("[Jucer2Cmake] Error while opening file %s: %w", jucerFilePath, err)
	}

	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("[Jucer2Cmake] Error while parsing XML: %w", err)
	}
	normalizeRawJucerProject(&raw, pluginFormats)

	proj = &PluginProject{}

	// apply minimum required metadata
	proj.Name = getString(raw.Name, "MOSAC")
	proj.Version = getString(raw.Version, "1.0.0")
	proj.CompanyName = getString(raw.CompanyName, "LIM")
	proj.PluginManufacturerCode = getString(raw.PluginManufacturerCode, "LIM!")
	proj.PluginManufacturer = getString(raw.PluginManufacturer, "LIM")
	pluginCode := getString(raw.PluginCode, "Lim0")
	if pluginCode == "Lim0" && raw.ID != nil {
		pluginCode = getPluginCodeFromUID(*raw.ID)
	}
	proj.PluginCode = pluginCode
	proj.PluginDesc = getString(raw.PluginDesc, "insert here italian plugin description")
	proj.PluginName = getString(raw.PluginName, proj.Name)
	proj.CompanyEmail = getString(raw.CompanyEmail, "lim@di.unimi.it")
	proj.CompanyWebsite = getString(raw.CompanyWebsite, "https://www.lim.di.unimi.it/")
	proj.CompanyCopyright = getString(raw.CompanyCopyright, proj.CompanyEmail)

	// formatting
	proj.CompanyName = strings.ReplaceAll(proj.CompanyName, " ", "-")
	proj.Name = strings.ReplaceAll(proj.Name, " ", "-")
	proj.PluginName = strings.ReplaceAll(proj.PluginName, " ", "-")
	if !strings.HasPrefix(proj.CompanyWebsite, "https://") {
		proj.CompanyWebsite = "https://" + proj.CompanyWebsite
	}

	// pluginFormats
	if len(pluginFormats) > 0 {
		proj.PluginFormats = pluginFormats
	} else if raw.PluginFormats == nil {
		proj.PluginFormats = []string{"Standalone", "VST3", "AU", "LV2", "Unity"}
	} else {
		formats := strings.Split(*raw.PluginFormats, ",")
		formatMapping := map[string]string{
			"buildVST3":       "VST3",
			"buildAU":         "AU",
			"buildAAX":        "AAX",
			"buildStandalone": "Standalone",
			"buildLV2":        "LV2",
			"buildUnity":      "Unity",
		}
		for _, f := range formats {
			if mapped, ok := formatMapping[f]; ok {
				proj.PluginFormats = append(proj.PluginFormats, mapped)
			}
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
	vst3Str := getString(raw.PluginVST3Category, "Fx")
	proj.PluginVST3Category = strings.Split(vst3Str, ",")

	// AAX category
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

func generateCMakeLists(proj *PluginProject, cmakeOutputPath string) error {
	cmakeOutDir := filepath.Dir(cmakeOutputPath)
	if err := os.MkdirAll(cmakeOutDir, 0755); err != nil {
		return fmt.Errorf("[Jucer2Cmake] Error creating output directory: %w", err)
	}

	file, err := os.Create(cmakeOutputPath)
	if err != nil {
		return fmt.Errorf("[Jucer2Cmake] Error creating CMakeLists.txt: %w", err)
	}
	defer file.Close()

	var b strings.Builder

	b.WriteString("cmake_minimum_required(VERSION 3.22)\n\n")
	b.WriteString(fmt.Sprintf("project(%s VERSION %s)\nset(CMAKE_CXX_STANDARD 17)\nset(CMAKE_POSITION_INDEPENDENT_CODE ON)\n", proj.Name, proj.Version))

	osName := `
function(get_os_name OUTPUT_VARIABLE)
  if(CMAKE_SYSTEM_NAME STREQUAL "Darwin")
    set(${OUTPUT_VARIABLE} "MacOS" PARENT_SCOPE)
  elseif(CMAKE_SYSTEM_NAME STREQUAL "Linux")
    set(${OUTPUT_VARIABLE} "Linux" PARENT_SCOPE)
  elseif(CMAKE_SYSTEM_NAME STREQUAL "Windows")
    set(${OUTPUT_VARIABLE} "Windows" PARENT_SCOPE)
  else()
    set(${OUTPUT_VARIABLE} "null" PARENT_SCOPE)
  endif()
endfunction()
`
	b.WriteString(osName)
	b.WriteString("\n")
	b.WriteString("get_os_name(CURRENT_OS)\n")
	b.WriteString("add_subdirectory(\"${JUCE_PATH}\" \"${PROJECT_DIR}/build/${CURRENT_OS}/JUCE_build\")\n\n")

	// juce_add_plugin
	b.WriteString(fmt.Sprintf("juce_add_plugin(%s\n", proj.PluginName))
	b.WriteString(fmt.Sprintf("\tVERSION %s\n", proj.Version))
	b.WriteString(fmt.Sprintf("\tCOMPANY_NAME \"%s\"\n", proj.CompanyName))

	// Metadati opzionali
	if proj.CompanyEmail != "" {
		b.WriteString(fmt.Sprintf("\tCOMPANY_EMAIL \"%s\"\n", proj.CompanyEmail))
	}
	if proj.CompanyWebsite != "" {
		b.WriteString(fmt.Sprintf("\tCOMPANY_WEBSITE \"%s\"\n", proj.CompanyWebsite))
	}
	if proj.CompanyCopyright != "" {
		b.WriteString(fmt.Sprintf("\tCOMPANY_COPYRIGHT \"%s\"\n", proj.CompanyCopyright))
	}
	if proj.PluginDesc != "" {
		b.WriteString(fmt.Sprintf("\tDESCRIPTION \"%s\"\n", proj.PluginDesc))
	}
	b.WriteString(fmt.Sprintf("\tPLUGIN_MANUFACTURER \"%s\"\n", proj.PluginManufacturer))
	b.WriteString(fmt.Sprintf("\tPLUGIN_MANUFACTURER_CODE %s\n", proj.PluginManufacturerCode))
	b.WriteString(fmt.Sprintf("\tPLUGIN_CODE %s\n", proj.PluginCode))
	b.WriteString(fmt.Sprintf("\tPLUGIN_NAME \"%s\"\n", proj.PluginName))
	b.WriteString(fmt.Sprintf("\tFORMATS %s\n", strings.Join(proj.PluginFormats, " ")))
	b.WriteString(fmt.Sprintf("\tIS_SYNTH %s\n", proj.IsSynth))
	b.WriteString(fmt.Sprintf("\tNEEDS_MIDI_INPUT %s\n", proj.WantsMidiInput))
	b.WriteString(fmt.Sprintf("\tNEEDS_MIDI_OUTPUT %s\n", proj.ProducesMidiOut))
	b.WriteString(fmt.Sprintf("\tIS_MIDI_EFFECT %s\n", proj.IsMidiEffect))
	b.WriteString(fmt.Sprintf("\tEDITOR_WANTS_KEYBOARD_FOCUS %s\n", proj.EditorRequiresKeys))
	b.WriteString(fmt.Sprintf("\tPRODUCT_NAME \"%s\"\n", proj.PluginName))
	b.WriteString("\tCOPY_PLUGIN_AFTER_BUILD FALSE")

	// LV2, VST3, AAX, AU
	if slices.Contains(proj.PluginFormats, "LV2") {
		b.WriteString(fmt.Sprintf("\n\tLV2URI \"%s/%s\"", proj.CompanyWebsite, proj.PluginName))
	}
	if slices.Contains(proj.PluginFormats, "VST3") {
		b.WriteString(fmt.Sprintf("\n\tVST3_CATEGORIES \"%s\"\n\tVST3_AUTO_MANIFEST FALSE", strings.Join(proj.PluginVST3Category, "\" \"")))
	}
	if slices.Contains(proj.PluginFormats, "AAX") {
		b.WriteString(fmt.Sprintf("\n\tAAX_CATEGORY %s", strings.Join(proj.PluginAAXCategory, " ")))
	}
	if slices.Contains(proj.PluginFormats, "AU") {
		b.WriteString(fmt.Sprintf("\n\tAU_MAIN_TYPE %s", proj.PluginAUMainType))
	}
	b.WriteString(")\n\n")

	b.WriteString(fmt.Sprintf("juce_generate_juce_header(%s)\n\n", proj.PluginName))

	// cross-compilation manifest for Windows
	manifestStr := `if (CMAKE_SYSTEM_NAME STREQUAL "Windows")
  set(STANDALONE_TARGET "--PLUGINNAME--_Standalone")
  set(VST3_TARGET "--PLUGINNAME--_VST3")
      
  set(MANIFEST_PATH "${CMAKE_CURRENT_BINARY_DIR}/WinManifest.xml")
  file(WRITE "${MANIFEST_PATH}" [=[<?xml version="1.0" encoding="UTF-8"?>
  <assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
    <trustInfo>
      <security>
        <requestedPrivileges>
          <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
        </requestedPrivileges>
      </security>
    </trustInfo>
    <dependency>
      <dependentAssembly>
        <assemblyIdentity type="Win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>
      </dependentAssembly>
    </dependency>
  </assembly>]=])

  if (TARGET ${STANDALONE_TARGET})
    set(RC_PATH_EXE "${CMAKE_CURRENT_BINARY_DIR}/WinResource_EXE.rc")
    file(WRITE "${RC_PATH_EXE}" "1 24 \"${MANIFEST_PATH}\"\n")
    target_sources(${STANDALONE_TARGET} PRIVATE "${RC_PATH_EXE}")
  endif()

  if (TARGET ${VST3_TARGET})
    set(RC_PATH_DLL "${CMAKE_CURRENT_BINARY_DIR}/WinResource_DLL.rc")
    file(WRITE "${RC_PATH_DLL}" "2 24 \"${MANIFEST_PATH}\"\n")
    target_sources(${VST3_TARGET} PRIVATE "${RC_PATH_DLL}")
  endif()
endif()

`
	b.WriteString(strings.ReplaceAll(manifestStr, "--PLUGINNAME--", proj.PluginName))

	// source files + external libraries (if present)
	libSources := ""
	if len(proj.LibrarySources) > 0 {
		libSources = "\n\t"
		libSources += strings.Join(proj.LibrarySources, "\n\t")
		libSources += "\n"
	}
	b.WriteString(fmt.Sprintf(`
file(GLOB_RECURSE %s_SOURCES CONFIGURE_DEPENDS
	${CMAKE_CURRENT_SOURCE_DIR}/Source/*.cpp
	${CMAKE_CURRENT_SOURCE_DIR}/Source/*.c
	${CMAKE_CURRENT_SOURCE_DIR}/Source/*.h
)
target_sources(%s PRIVATE ${%s_SOURCES}%s)

`, proj.PluginName, proj.PluginName, proj.PluginName, libSources))

	// headers
	b.WriteString(fmt.Sprintf(`
file(GLOB_RECURSE ALL_DIRS LIST_DIRECTORIES true "${CMAKE_CURRENT_SOURCE_DIR}/Source/*")
foreach(DIR ${ALL_DIRS})
	if(IS_DIRECTORY ${DIR})
		target_include_directories(%s PRIVATE ${DIR})
	endif()
endforeach()

`, proj.PluginName))

	// Libraries folder
	b.WriteString(fmt.Sprintf("target_include_directories(%s SYSTEM PRIVATE \"${CMAKE_CURRENT_SOURCE_DIR}/Libraries\")\n\n", proj.PluginName))

	// compile definitions
	b.WriteString(fmt.Sprintf("target_compile_definitions(%s\n\tPUBLIC\n", proj.PluginName))
	for _, define := range proj.Defines {
		b.WriteString(fmt.Sprintf("\t\t%s=1\n", define))
	}
	b.WriteString("\t\tJUCE_WEB_BROWSER=0\n\t\tJUCE_USE_CURL=0\n\t\tJUCE_VST3_CAN_REPLACE_VST2=0)\n\n")

	// assets files (binary data)
	hasBinaryData := proj.IncludeBinaryInJuceHeader == 1 && len(proj.AssetFiles) > 0
	if hasBinaryData {
		b.WriteString(fmt.Sprintf("juce_add_binary_data(%s SOURCES\n", proj.BinaryDataNamespace))
		for _, asset := range proj.AssetFiles {
			b.WriteString(fmt.Sprintf("\t\"%s\"\n", asset))
		}
		b.WriteString(")\n\n")
	}

	// modules and binary data linking
	b.WriteString(fmt.Sprintf("target_link_libraries(%s\n", proj.PluginName))
	if len(proj.Modules) > 0 || hasBinaryData {
		b.WriteString("\tPRIVATE\n")
		if len(proj.Modules) > 0 {
			b.WriteString(fmt.Sprintf("\t\t%s\n", strings.Join(proj.Modules, "\n\t\t")))
		}
		if hasBinaryData {
			b.WriteString(fmt.Sprintf("\t\t%s\n", proj.BinaryDataNamespace))
		}
	}
	b.WriteString("\tPUBLIC\n\t\tjuce::juce_recommended_config_flags\n\t\tjuce::juce_recommended_lto_flags\n\t\tjuce::juce_recommended_warning_flags)\n\n")

	// compile options (suppress warnings)
	b.WriteString(fmt.Sprintf(`target_compile_options(%s PRIVATE
	$<$<CXX_COMPILER_ID:GNU>:-w>
	$<$<CXX_COMPILER_ID:GNU>:-fpermissive>
	$<$<OR:$<CXX_COMPILER_ID:Clang>,$<CXX_COMPILER_ID:AppleClang>>:-w>
	$<$<OR:$<CXX_COMPILER_ID:Clang>,$<CXX_COMPILER_ID:AppleClang>>:-Wno-deprecated-declarations>
)`, proj.PluginName))

	_, err = file.WriteString(b.String())
	return err
}

// Creates a CMakeLists.txt file from a JUCE .jucer (Basic Audio Plugin) project file.
// It takes the path to the project directory and a list of plugin formats to include in the CMakeLists.txt file (it overrides the formats specified in the .jucer file if provided).
// Returns a pointer to the parsed PluginProject struct and an error if any.
func Jucer2Cmake(ProjectPath string, PluginFormats []string) (*PluginProject, error) {
	filePath, err := getJucerFilePath(ProjectPath)
	if err != nil {
		return nil, err
	}

	proj, err := parseJucerFile(filePath, PluginFormats)
	if err != nil {
		return nil, err
	}

	cmakeOutputPath := filepath.Join(ProjectPath, "CMakeLists.txt")
	err = generateCMakeLists(proj, cmakeOutputPath)
	if err != nil {
		return nil, err
	}

	return proj, nil
}
