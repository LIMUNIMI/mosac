package batchMaker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mosac/internal/builders"
)

// scans the provided directory for projects and generates a batch.csv file.
func CreateBatchCSV(analysisDir string, defaultJuce int, outputDir, buildConfig, systems, formats string) error {
	analysisPath, err := filepath.Abs(analysisDir)
	if err != nil {
		return fmt.Errorf("error resolving absolute path for analysis directory: %v", err)
	}

	outPath, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("error resolving absolute path for output directory: %v", err)
	}

	entries, err := os.ReadDir(analysisPath)
	if err != nil {
		return fmt.Errorf("error reading analysis directory: %v", err)
	}

	var projectPaths []string
	for _, entry := range entries {
		if entry.IsDir() {
			subDirPath := filepath.Join(analysisPath, entry.Name())
			projectPaths = append(projectPaths, subDirPath)
		}
	}

	if len(projectPaths) == 0 {
		return fmt.Errorf("no project directories found in %s", analysisPath)
	}

	info, err := os.Stat(outPath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(outPath, 0755)
		if err != nil {
			return fmt.Errorf("error creating output directory: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("error checking the output directory: %v", err)
	} else if !info.IsDir() {
		return fmt.Errorf("the specified output path '%s' is not a directory", outPath)
	}

	batchFilePath := filepath.Join(outPath, "batch.csv")
	file, err := os.Create(batchFilePath)
	if err != nil {
		return fmt.Errorf("error creating the file batch.csv: %v", err)
	}
	defer file.Close()

	sysCsvList := strings.ReplaceAll(systems, ",", ";")
	fmtCsvList := strings.ReplaceAll(formats, ",", ";")

	defaultJuceStr := fmt.Sprintf("%d", defaultJuce)

	for _, projPath := range projectPaths {
		juceClm := strings.TrimSpace(defaultJuceStr)

		// if the .jucer file in userNotes section has a juce:n line, use it. use the defaultJuce value otherwise.
		juceVer := builders.ExtractJuceVersionFromUserNotes(projPath)
		if juceVer == -1 {
			fmt.Printf("    Warning: Could not determine JUCE version for project at %s. Using default JUCE version: %d\n", projPath, defaultJuce)
		} else {
			juceClm = fmt.Sprintf("%d", juceVer)
		}

		line := fmt.Sprintf("%s,%s,%s,%s,%s\n", projPath, juceClm, buildConfig, sysCsvList, fmtCsvList)

		if _, err := file.WriteString(line); err != nil {
			return fmt.Errorf("error while writing to batch.csv: %v", err)
		}
	}

	fmt.Printf("\n    Successfully created batch file at: %s\n", batchFilePath)
	fmt.Printf("    Found %d projects.\n", len(projectPaths))

	return nil
}
