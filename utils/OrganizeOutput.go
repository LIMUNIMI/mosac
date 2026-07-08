package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"io/fs"
	"encoding/json"
	"strings"
)

type PluginMetadata struct {
	PluginName      string   `json:"pluginName"`
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



func writeInfoToJson(proj *PluginProject, outputDir string, buildForOS, pluginFormats []string) error {
	file, err := os.Create(filepath.Join(outputDir, fmt.Sprintf("%s.json", proj.PluginName)))
	if err != nil {return fmt.Errorf("[Output] Error creating JSON file: %w", err)}
	defer file.Close()

	jsonData := PluginMetadata{
		PluginName:      strings.ReplaceAll(proj.PluginName, "_", " "),
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




func OrganizeOutput(proj *PluginProject, projDir, outputDir, buildType string, buildForOS, pluginFormats []string) error {
	contentDir := filepath.Join(outputDir, proj.PluginName)
	var (
		err error
		toBeRemoved []string
	)

	if err = os.MkdirAll(contentDir, os.ModePerm); err != nil {return fmt.Errorf("[Output] Error occurred while creating output directory: %w", err)}

	for _, os := range buildForOS {
		for _, format := range pluginFormats {
			srcDir := filepath.Join(projDir, "build", os, buildType, proj.PluginName + "_artefacts", buildType, format)
			dstDir := filepath.Join(contentDir, os, buildType, format)

			if err := copyDirectory(srcDir, dstDir); err != nil {return fmt.Errorf("[Output] Error occurred while copying %s to %s: %w", srcDir, dstDir, err)}
		}
	}

	toBeRemoved = append(toBeRemoved, 
		filepath.Join(contentDir, "Windows", buildType,"Standalone", fmt.Sprintf("%s.exe.manifest", proj.PluginName)),
		filepath.Join(contentDir, "Windows", buildType,"LV2", fmt.Sprintf("%s.lib", proj.PluginName)),
		filepath.Join(contentDir, "Windows", buildType,"VST3", fmt.Sprintf("%s.lib", proj.PluginName)),
	)

	removeFiles(toBeRemoved)

	if err := writeInfoToJson(proj, outputDir, buildForOS, pluginFormats); err != nil {return err}

	return nil
}