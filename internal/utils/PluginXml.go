package utils

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type RawFile struct {
	File     string `xml:"file,attr"`
	Resource string `xml:"resource,attr"`
}

// group of files, can contain subgroups
type RawGroup struct {
	Files  []RawFile  `xml:"FILE"`
	Groups []RawGroup `xml:"GROUP"`
}

type RawModule struct {
	ID string `xml:"id,attr"`
}

type RawJucerProject struct {
	XMLName xml.Name `xml:"JUCERPROJECT"`
	ID      *string  `xml:"id,attr"`
	Name    *string  `xml:"name,attr"`
	Version *string  `xml:"version,attr"`

	PluginName             *string `xml:"pluginName,attr"`
	PluginManufacturer     *string `xml:"pluginManufacturer,attr"`
	PluginManufacturerCode *string `xml:"pluginManufacturerCode,attr"`
	PluginCode             *string `xml:"pluginCode,attr"`
	PluginDesc             *string `xml:"pluginDesc,attr"`
	PluginFormats          *string `xml:"pluginFormats,attr"`
	PluginVST3Category     *string `xml:"pluginVST3Category,attr"`
	PluginAAXCategory      *string `xml:"pluginAAXCategory,attr"`
	PluginAUMainType       *string `xml:"pluginAUMainType,attr"`
	UserNotes              *string `xml:"userNotes,attr"`

	PluginCharacteristicsValue *string `xml:"pluginCharacteristicsValue,attr"`
	IncludeBinaryInJuceHeader  *int    `xml:"includeBinaryInJuceHeader,attr"`
	BinaryDataNamespace        *string `xml:"binaryDataNamespace,attr"`

	CompanyName      *string `xml:"companyName,attr"`
	CompanyEmail     *string `xml:"companyEmail,attr"`
	CompanyWebsite   *string `xml:"companyWebsite,attr"`
	CompanyCopyright *string `xml:"companyCopyright,attr"`

	Defines   *string     `xml:"defines,attr"`
	Modules   []RawModule `xml:"MODULES>MODULE"`
	MainGroup RawGroup    `xml:"MAINGROUP"`
}

type PluginProject struct {
	ID      string
	Name    string
	Version string

	PluginName             string
	PluginManufacturer     string
	PluginManufacturerCode string
	PluginCode             string
	PluginDesc             string
	PluginFormats          []string
	PluginVST3Category     []string
	PluginAAXCategory      []string
	PluginAUMainType       string
	UserNotes              string

	Owner_json  string // used in pluginName.json metadata report file
	Rating_json int    // used in pluginName.json metadata report file

	EditorRequiresKeys        string
	IsMidiEffect              string
	IsSynth                   string
	WantsMidiInput            string
	ProducesMidiOut           string
	IncludeBinaryInJuceHeader int
	BinaryDataNamespace       string

	CompanyName      string
	CompanyEmail     string
	CompanyWebsite   string
	CompanyCopyright []string // array because it can contain multiple authors names.

	Defines        []string
	Modules        []string
	AssetFiles     []string
	LibrarySources []string
}

func getString(ptr *string, defaultVal string) string {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

func getInt(ptr *int, defaultVal int) int {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

// pluginCode generated from the project UID using the same logic as Projucer
func getPluginCodeFromUID(uid string) string {
	runes := []rune(uid[0:4])
	runes[0] = unicode.ToUpper(runes[0])
	for i := 1; i < 4; i++ {
		runes[i] = unicode.ToLower(runes[i])
	}
	return string(runes)
}

// searches "owner:....." and "rating:n" (with 0<n<6) in separate lines in the UserNotes and returns the parsed values
func parseUserNotes(userNotes string) (owner string, rating int) {
	owner = "default.owner@email.com"
	rating = -1

	// use a scanner to read the userNotes line by line
	scanner := bufio.NewScanner(strings.NewReader(userNotes))
	for scanner.Scan() {
		line := scanner.Text()

		if rest, found := strings.CutPrefix(line, "owner:"); found {
			owner = strings.TrimSpace(rest)
		} else if rest, found := strings.CutPrefix(line, "rating:"); found {
			ratingStr := strings.TrimSpace(rest)
			parsedRating, err := strconv.Atoi(ratingStr)
			if err != nil || parsedRating < 0 || parsedRating > 5 {
				fmt.Printf("-- non valid <rating> in UserNotes found -> rating = -1.\n\n")
				rating = -1
			} else {
				rating = parsedRating
			}
		}
	}

	return owner, rating
}
