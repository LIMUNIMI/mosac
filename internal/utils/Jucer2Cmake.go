package utils

import (
	"fmt"
	"os"
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

// Creates a CMakeLists.txt file from a given pluginProject struct
//! NOTE: in theory, AAX format is not in proj.PluginFormats array
// Returns an error if any.
func Jucer2Cmake(cmakeOutputPath string, proj *PluginProject) error {
	file, err := os.Create(cmakeOutputPath)
	if err != nil {
		return fmt.Errorf("[generateCMakeLists] Error creating CMakeLists.txt: %w", err)
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
	b.WriteString(fmt.Sprintf("\tPLUGIN_NAME \"%s\"\n", proj.PluginName))
	b.WriteString(fmt.Sprintf("\tPLUGIN_MANUFACTURER \"%s\"\n", proj.PluginManufacturer))
	b.WriteString(fmt.Sprintf("\tPLUGIN_MANUFACTURER_CODE %s\n", proj.PluginManufacturerCode))
	b.WriteString(fmt.Sprintf("\tPLUGIN_CODE %s\n", proj.PluginCode))
	b.WriteString(fmt.Sprintf("\tDESCRIPTION \"%s\"\n", proj.PluginDesc))
	b.WriteString(fmt.Sprintf("\tCOMPANY_NAME \"%s\"\n", proj.CompanyName))
	b.WriteString(fmt.Sprintf("\tCOMPANY_EMAIL \"%s\"\n", proj.CompanyEmail))
	b.WriteString(fmt.Sprintf("\tCOMPANY_WEBSITE \"%s\"\n", proj.CompanyWebsite))
	b.WriteString(fmt.Sprintf("\tCOMPANY_COPYRIGHT \"%s\"\n", strings.Join(proj.CompanyCopyright, ", ")))
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
