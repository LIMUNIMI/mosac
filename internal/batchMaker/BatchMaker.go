package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Error: missing arguments.")
		fmt.Println("Usage: go run BatchMaker.go <dirPath> <JucePath> <batchOutputPath>")
		fmt.Println("If a project folder contains 'mosac.conf', the JUCE path column is left empty and the build will resolve the JUCE version automatically.")
		return
	}

	rawAnalysisPath := os.Args[1]
	rawJucePath := os.Args[2]
	rawOutputPath := os.Args[3]

	analysisPath, err := filepath.Abs(rawAnalysisPath)
	if err != nil {fmt.Printf("Error while making absolute path for analysis: %v\n", err);return}
	jucePath, err := filepath.Abs(rawJucePath)
	if err != nil {fmt.Printf("Error while making absolute path for JUCE: %v\n", err);return}
	outputPath, err := filepath.Abs(rawOutputPath)
	if err != nil {fmt.Printf("Error while making absolute path for output: %v\n", err);return}

	entries, err := os.ReadDir(analysisPath)
	if err != nil {fmt.Printf("Error while reading analysis directory: %v\n", err);return}

	var ProjectPaths []string
	for _, entry := range entries {
		if entry.IsDir() {
			subDirPath := filepath.Join(analysisPath, entry.Name())
			ProjectPaths = append(ProjectPaths, subDirPath)
		}
	}

	info, err := os.Stat(outputPath)
	if os.IsNotExist(err) || !info.IsDir() {fmt.Printf("Error: the specified output directory '%s' does not exist or is not a folder.\n", outputPath)
		return
	} else if err != nil {
		fmt.Printf("Error while checking the output directory: %v\n", err)
		return
	}

	batchFilePath := filepath.Join(outputPath, "batch.csv")
	file, err := os.Create(batchFilePath)
	if err != nil {fmt.Printf("Error while creating the file %s: %v\n", "batch.csv", err);return}

	for _, projPath := range ProjectPaths {
		juceClm := jucePath // if dir contains mosac.conf, csv's JUCE path is emmpty
		if info, err := os.Stat(filepath.Join(projPath, "mosac.conf")); err == nil && !info.IsDir() {juceClm = ""}

		line := fmt.Sprintf("%s,%s,Release,Linux;Windows;MacOS,Standalone;VST3;AU;LV2;Unity\n", projPath, strings.TrimSpace(juceClm))
		
		_, err := file.WriteString(line)
		if err != nil {
			fmt.Printf("Error while writing to the file: %v\n", err)
			file.Close()
			return
		}
	}

	err = file.Close()
	if err != nil {
		fmt.Printf("Error during the closing of the file: %v\n", err)
		return
	}

	fmt.Printf("Batch file %s created successfully.\n", "batch.csv")
}