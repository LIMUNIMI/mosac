package project

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	line = strings.TrimPrefix(line, "JUCE-")
	line = strings.TrimPrefix(line, "JUCE")

	majorVersion, err := strconv.Atoi(line)
	if err != nil || majorVersion < 1 {
		return "", fmt.Errorf("[MosacConfig] invalid JUCE version in mosac.conf: %s", line)
	}

	return "JUCE" + strconv.Itoa(majorVersion), nil
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
	majorVersion, err := strconv.Atoi(version)
	if err != nil || majorVersion < 1 {
		return "", fmt.Errorf("[MOSAC] unsupported JUCE version: %s", version)
	}
	version = strconv.Itoa(majorVersion)

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

// accepts either an installed JUCE version or an existing directory path.
func ResolveJuceDir(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("[MOSAC] JUCE version or directory path is empty")
	}

	if info, err := os.Stat(input); err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("[MOSAC] JUCE path is not a directory: %s", input)
		}
		return filepath.Abs(input)
	}

	return ResolveJuceDirFromVersion(input)
}
