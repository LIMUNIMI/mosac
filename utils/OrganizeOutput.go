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
	EnglishDesc     string   `json:"englishDesc"`
	DownloadLink    string   `json:"downloadLink"`
	DemoImageLink   string   `json:"demoImageLink"`
	Authors         string   `json:"authors"`
	Email           string   `json:"email"`
	TargetOS        []string `json:"targetOS"`
	CompiledFormats []string `json:"compiledFormats"`
	FxCategory      []string `json:"FxCategory"`
}


// copies .pdf and .md files from the src directory to the dest directory.
func copyDocFiles(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {return fmt.Errorf("[Output] Errore lettura directory %s: %w", srcDir, err)}

	for _, entry := range entries {
		if entry.IsDir() {continue}

		// either .pdf and .PDF or .md and .MD or .txt
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".pdf" || ext == ".md" || ext == ".txt" || ext == ".png" || ext == ".jpg" {
			srcPath := filepath.Join(srcDir, entry.Name())
			dstPath := filepath.Join(destDir, entry.Name())

			data, err := os.ReadFile(srcPath)
			if err != nil {return fmt.Errorf("[Output] Errore lettura documento %s: %w", entry.Name(), err)}

			if err := os.WriteFile(dstPath, data, 0644); err != nil {return fmt.Errorf("[Output] Errore scrittura documento %s: %w", entry.Name(), err)}
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
		if err != nil {return fmt.Errorf("errore lettura file %s: %w", path, err)}

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
func writeInfoToJson(proj *PluginProject, outputDir string, buildForOS, pluginFormats []string) error {
	file, err := os.Create(filepath.Join(outputDir, fmt.Sprintf("%s.json", proj.PluginName)))
	if err != nil {return fmt.Errorf("[Output] Error creating JSON file: %w", err)}
	defer file.Close()

	jsonData := PluginMetadata{
		PluginName:      strings.ReplaceAll(proj.PluginName, "_", " "),
		PluginVersion:   proj.Version,
		PluginDesc:      proj.PluginDesc,
		EnglishDesc:		 "insert here english plugin description",
		DownloadLink:    "insert download link here",
		DemoImageLink:   "insert demo image link here",
		Authors:         "insert authors here",
		Email:           "insert email here",
		TargetOS:        buildForOS,
		CompiledFormats: pluginFormats,
		FxCategory:      proj.PluginVST3Category,
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "	")

	err = encoder.Encode(jsonData)
	if err != nil {return fmt.Errorf("[Output] Error writing JSON data: %w", err)}

	return nil
}



// organizes the output of the plugin build process by copying the relevant files to a structured output directory and creating a JSON metadata file.
func OrganizeOutput(proj *PluginProject, projDir, outputDir, buildType string, buildForOS, pluginFormats []string) error {
	contentDir := filepath.Join(outputDir, proj.PluginName)
	var (
		err error
		toBeRemoved []string
	)

	filteredFormats := make([]string, 0, len(pluginFormats))
	if !slices.Contains(buildForOS, "MacOS") && slices.Contains(pluginFormats, "AU") {
		for _, f := range pluginFormats {
			if f != "AU" {filteredFormats = append(filteredFormats, f)}
		}
	} else {
		filteredFormats = pluginFormats
	}

	if err = os.MkdirAll(contentDir, os.ModePerm); err != nil {return fmt.Errorf("[Output] Error occurred while creating output directory: %w", err)}

	for _, os := range buildForOS {
		for _, format := range filteredFormats {
			srcDir := filepath.Join(projDir, "build", os, buildType, proj.PluginName + "_artefacts", buildType, format)
			dstDir := filepath.Join(contentDir, os, buildType, format)

			copyDirectory(srcDir, dstDir)
		}
	}

	toBeRemoved = append(toBeRemoved, 
		filepath.Join(contentDir, "Windows", buildType,"Standalone", fmt.Sprintf("%s.exe.manifest", proj.PluginName)),
		filepath.Join(contentDir, "Windows", buildType,"LV2", fmt.Sprintf("%s.lib", proj.PluginName)),
		filepath.Join(contentDir, "Windows", buildType,"VST3", fmt.Sprintf("%s.lib", proj.PluginName)),
	)

	// remove manifest and .lib files that are not needed in the output directory
	removeFiles(toBeRemoved)

	if err := writeInfoToJson(proj, outputDir, buildForOS, filteredFormats); err != nil {return err}
	if err := copyDocFiles(projDir, contentDir); err != nil {return err}
	return nil
}