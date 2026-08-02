package utils

import (
	"slices"
	"fmt"
	"os"
	"path/filepath"
	"io/fs"
	"encoding/json"
	"strings"
)

type PluginMetadata struct {
	PluginName      string   `json:"pluginName"`
	PluginVersion   string   `json:"pluginVersion"`
	PluginDesc      string   `json:"pluginDesc"`
	FxCategory      []string `json:"FxCategory"`
	
	EnglishDesc     string   `json:"englishDesc"`
	Authors         []string   `json:"authors"`
	Email           []string   `json:"email"`
	Url             string   `json:"url"`

	TargetOS        []string `json:"targetOS"`
	CompiledFormats []string `json:"compiledFormats"`
}


// copies .pdf and .md files from the src directory to the dest directory.
func copyDocFiles(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {return fmt.Errorf("[Output] Error reading directory %s: %w", srcDir, err)}

	for _, entry := range entries {
		if entry.IsDir() {continue}

		// either .pdf and .PDF or .md and .MD or .txt
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".pdf" || ext == ".md" || (ext == ".txt" && entry.Name() != "CMakeLists.txt") || ext == ".png" || ext == ".jpg" {
			srcPath := filepath.Join(srcDir, entry.Name())
			dstPath := filepath.Join(destDir, entry.Name())

			data, err := os.ReadFile(srcPath)
			if err != nil {return fmt.Errorf("[Output] Error reading document %s: %w", entry.Name(), err)}

			if err := os.WriteFile(dstPath, data, 0644); err != nil {return fmt.Errorf("[Output] Error writing document %s: %w", entry.Name(), err)}
		}
	}
	return nil
}


// copies the contents of the source directory to the destination directory.
func copyDirectory(src, dst string) error {
	srcFS := os.DirFS(src)

	return fs.WalkDir(srcFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {return err}

		targetPath := filepath.Join(dst, path)

		if d.IsDir() {
			info, _ := d.Info()
			return os.MkdirAll(targetPath, info.Mode())
		}

		data, err := fs.ReadFile(srcFS, path)
		if err != nil {return fmt.Errorf("[Output] Error reading file %s: %w", path, err)}

		info, _ := d.Info()
		return os.WriteFile(targetPath, data, info.Mode())
	})
}



func removeFiles(toBeRemoved []string) {
	for _, file := range toBeRemoved {
		os.Remove(file)
	}
}


// creates a JSON file containing metadata about the current plugin build.
func writeInfoToJson(proj *PluginProject, config *MosacConfig, outputDir string, buildForOS, pluginFormats []string) error {
	file, err := os.Create(filepath.Join(outputDir, fmt.Sprintf("%s.json", proj.PluginName)))
	if err != nil {return fmt.Errorf("[Output] Error creating JSON file: %w", err)}
	defer file.Close()

	englishDesc := "insert here english plugin description"
	authors := []string{"author1", "author2"}
	email := []string{"email1@example.com", "email2@example.com"}
	url := ""

	if config != nil {
		englishDesc = config.EnglishDesc
		authors = config.Authors
		email = config.Emails
		url = config.Url
	}

	jsonData := PluginMetadata{
		PluginName:      strings.ReplaceAll(proj.PluginName, "_", " "),
		PluginVersion:   proj.Version,
		PluginDesc:      proj.PluginDesc,
		FxCategory:      proj.PluginVST3Category,
		
		EnglishDesc:     englishDesc,
		Authors:         authors,
		Email:           email,
		Url:						 url,

		TargetOS:        buildForOS,
		CompiledFormats: pluginFormats,
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "	")

	err = encoder.Encode(jsonData)
	if err != nil {return fmt.Errorf("[Output] Error writing JSON data: %w", err)}

	return nil
}



// copies the builded pluginFormats to the outputDir and creates a JSON metadata file.
//if copyAll parameter is set to true, every plugin format of every OS gets copied regardless of the output code.
func OrganizeOutput(proj *PluginProject, config *MosacConfig, projDir, outputDir, buildType string, buildForOS, pluginFormats []string, copyAll bool) error {
	contentDir := filepath.Join(outputDir, proj.PluginName)
	var (
		err error
		toBeRemoved []string
	)
	
	if err = os.MkdirAll(contentDir, os.ModePerm); err != nil {return fmt.Errorf("[Output] Error occurred while creating output directory: %w", err)}
	filteredFormats := make([]string, 0, len(pluginFormats))

	if copyAll {
		// copies Builds/MacOSX folder to outputDir/pluginName/MacOS
		macDir := filepath.Join(projDir, "Builds", "MacOSX", "build", buildType)
		copyDirectory(macDir, filepath.Join(contentDir, "MacOS", buildType))

		// copies build/Linux folder to outputDir/pluginName/Linux
		linuxDir := filepath.Join(projDir, "build", "Linux", buildType, proj.PluginName + "_artefacts", buildType)
		copyDirectory(linuxDir, filepath.Join(contentDir, "Linux", buildType))

		// copies build/Windows folder to outputDir/pluginName/Windows
		winDir := filepath.Join(projDir, "build", "Windows", buildType, proj.PluginName + "_artefacts", buildType)
		copyDirectory(winDir, filepath.Join(contentDir, "Windows", buildType))

		goto Final
	}


	if !slices.Contains(buildForOS, "MacOS") && slices.Contains(pluginFormats, "AU") {
		for _, f := range pluginFormats {
			if f != "AU" {filteredFormats = append(filteredFormats, f)}
		}
	} else {
		filteredFormats = pluginFormats
	}

	// if macOS is in buildForOS, copy to the output directory
	if slices.Contains(buildForOS, "MacOS") {
		srcDir := filepath.Join(projDir, "Builds", "MacOSX", "build", buildType) //check if this is correct
		copyDirectory(srcDir, filepath.Join(contentDir, "MacOS", buildType))
	}

	for _, os := range buildForOS {
		//skip MacOS since it is already copied above
		if os == "MacOS" {continue}

		for _, format := range filteredFormats {
			srcDir := filepath.Join(projDir, "build", os, buildType, proj.PluginName + "_artefacts", buildType, format)
			dstDir := filepath.Join(contentDir, os, buildType, format)

			copyDirectory(srcDir, dstDir)
		}
	}

// final stage (cleaning and copying default directories)
Final:
	// if "Presets" and "Installers" folder exists in the project directory, copy them to the output directory
	if info, err := os.Stat(filepath.Join(projDir, "Presets")); err == nil && info.IsDir() {
		copyDirectory(filepath.Join(projDir, "Presets"), filepath.Join(contentDir, "Presets"))
	}
	if info, err := os.Stat(filepath.Join(projDir, "Installers")); err == nil && info.IsDir() {
		copyDirectory(filepath.Join(projDir, "Installers"), filepath.Join(contentDir, "Installers"))
	}

	toBeRemoved = append(toBeRemoved, 
		filepath.Join(contentDir, "Windows", buildType,"Standalone", fmt.Sprintf("%s.exe.manifest", proj.PluginName)),
		filepath.Join(contentDir, "Windows", buildType,"LV2", fmt.Sprintf("%s.lib", proj.PluginName)),
		filepath.Join(contentDir, "Windows", buildType,"LV2", fmt.Sprintf("%s.lv2", proj.PluginName), fmt.Sprintf("%s.dll.manifest", proj.PluginName)),
		filepath.Join(contentDir, "Windows", buildType,"VST3", fmt.Sprintf("%s.lib", proj.PluginName)),
		filepath.Join(contentDir, "Windows", buildType,"VST3", fmt.Sprintf("%s.vst3", proj.PluginName), "Contents", "x86_64-win", fmt.Sprintf("%s.vst3.manifest", proj.PluginName)),
		filepath.Join(contentDir, "Windows", buildType,"Unity", fmt.Sprintf("%s.dll.manifest", proj.PluginName)),
		filepath.Join(contentDir, "MacOS", buildType, "juce_lv2_helper"),
		filepath.Join(contentDir, "MacOS", buildType, "juce_vst3_helper"),
		filepath.Join(contentDir, "MacOS", buildType, fmt.Sprintf("lib%s.a", proj.PluginName)),

		filepath.Join(contentDir, "Linux", buildType, fmt.Sprintf("lib%s_SharedCode.a", proj.PluginName)),
		filepath.Join(contentDir, "Windows", buildType, fmt.Sprintf("%s_SharedCode.lib", proj.PluginName)),
	)
	
	// remove manifest and .lib files that are not needed in the output directory (also removes AAX folder if it exists)
	removeFiles(toBeRemoved)
	os.RemoveAll(filepath.Join(contentDir, "Windows", buildType,"AAX"))

	if err := writeInfoToJson(proj, config, outputDir, buildForOS, filteredFormats); err != nil {return err}
	if err := copyDocFiles(projDir, contentDir); err != nil {return err}
	return nil
}