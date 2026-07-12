package utils

import (
	"encoding/xml"
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
	XMLName                    xml.Name `xml:"JUCERPROJECT"`
	Name                       *string  `xml:"name,attr"`
	Version                    *string  `xml:"version,attr"`

	PluginName                 *string  `xml:"pluginName,attr"`
	PluginManufacturer         *string  `xml:"pluginManufacturer,attr"`
	PluginManufacturerCode     *string  `xml:"pluginManufacturerCode,attr"`
	PluginCode                 *string  `xml:"pluginCode,attr"`
	PluginDesc                 *string  `xml:"pluginDesc,attr"`
	PluginFormats              *string  `xml:"pluginFormats,attr"`
	PluginVST3Category         *string  `xml:"pluginVST3Category,attr"`
	PluginAAXCategory          *string  `xml:"pluginAAXCategory,attr"`
	PluginAUMainType           *string  `xml:"pluginAUMainType,attr"`
	
	PluginCharacteristicsValue *string  `xml:"pluginCharacteristicsValue,attr"`
	IncludeBinaryInJuceHeader  *int     `xml:"includeBinaryInJuceHeader,attr"`
	BinaryDataNamespace        *string  `xml:"binaryDataNamespace,attr"`
	
	CompanyName                *string  `xml:"companyName,attr"`
	CompanyEmail               *string  `xml:"companyEmail,attr"`
	CompanyWebsite             *string  `xml:"companyWebsite,attr"`
	CompanyCopyright           *string  `xml:"companyCopyright,attr"`
	
	Defines                    *string     `xml:"defines,attr"`
	Modules                    []RawModule `xml:"MODULES>MODULE"`
	MainGroup                  RawGroup    `xml:"MAINGROUP"`
}

type PluginProject struct {
	Name                      string
	Version                   string
	
	PluginName                string
	PluginManufacturer        string
	PluginManufacturerCode    string
	PluginCode                string
	PluginDesc                string
	PluginFormats             []string
	PluginVST3Category        []string
	PluginAAXCategory         []string
	PluginAUMainType          string
	
	EditorRequiresKeys        string
	IsMidiEffect              string
	IsSynth                   string
	WantsMidiInput            string
	ProducesMidiOut           string
	IncludeBinaryInJuceHeader int
	BinaryDataNamespace       string
	
	CompanyName               string
	CompanyEmail              string
	CompanyWebsite            string
	CompanyCopyright          string
	
	Defines                   []string
	Modules                   []string
	AssetFiles                []string
	LibrarySources            []string
}

func getString(ptr *string, defaultVal string) string {
	if ptr != nil {return *ptr}
	return defaultVal
}

func getInt(ptr *int, defaultVal int) int {
	if ptr != nil {return *ptr}
	return defaultVal
}