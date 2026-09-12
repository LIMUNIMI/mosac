package project

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MosacConfig struct {
	JuceVersion string
	EnglishDesc string
	Authors     []string
	Emails      []string
	Url         string
}

func parseCommaSeparatedLine(line string) []string {
	parts := strings.Split(line, ",")
	values := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}

	return values
}

func parseJuceVersion(line string) (string, error) {
	line = strings.ToUpper(strings.TrimSpace(line))
	line = strings.ReplaceAll(line, " ", "")

	if line == "JUCE7" || line == "JUCE-7" {
		return "JUCE7", nil
	}
	if line == "JUCE8" || line == "JUCE-8" {
		return "JUCE8", nil
	}

	return "", fmt.Errorf("[MosacConfig] invalid JUCE version in mosac.conf: %s", line)
}

func LoadMosacConfig(projectDir string) (*MosacConfig, bool, error) {
	configPath := filepath.Join(projectDir, "mosac.conf")
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("[MosacConfig] error while opening %s: %w", configPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := make([]string, 0, 5)
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return nil, true, fmt.Errorf("[MosacConfig] error while reading %s: %w", configPath, err)
	}

	if len(lines) < 4 {
		return nil, true, fmt.Errorf("[MosacConfig] invalid mosac.conf: expected 4 lines, got %d", len(lines))
	}

	juceVersion, err := parseJuceVersion(lines[0])
	if err != nil {
		return nil, true, err
	}

	aux := ""
	if len(lines) == 5 {
		if lines[4] != "" {
			aux = lines[4]
		}
	}

	return &MosacConfig{
		JuceVersion: juceVersion,
		EnglishDesc: lines[1],
		Authors:     parseCommaSeparatedLine(lines[2]),
		Emails:      parseCommaSeparatedLine(lines[3]),
		Url:         aux,
	}, true, nil
}

func ResolveJuceDirFromVersion(version string) (string, error) {
	version = strings.TrimSpace(strings.ToUpper(version))
	version = strings.TrimPrefix(version, "JUCE-")
	version = strings.TrimPrefix(version, "JUCE")
	if version != "7" && version != "8" {
		return "", fmt.Errorf("[MOSAC] unsupported JUCE version: %s", version)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("[MOSAC] could not get user home directory: %w", err)
	}

	juceDir := filepath.Join(homeDir, ".mosac", "juce", version)
	info, err := os.Stat(juceDir)
	if err != nil {
		return "", fmt.Errorf("[MOSAC] could not resolve JUCE directory for version %s at %s: %w", version, juceDir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("[MOSAC] JUCE path is not a directory: %s", juceDir)
	}

	return filepath.Abs(juceDir)
}
