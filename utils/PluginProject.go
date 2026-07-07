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
	CompanyName                *string  `xml:"companyName,attr"`
	PluginManufacturerCode     *string  `xml:"pluginManufacturerCode,attr"`
	PluginManufacturer         *string  `xml:"pluginManufacturer,attr"`
	PluginCode                 *string  `xml:"pluginCode,attr"`
	PluginDesc                 *string  `xml:"pluginDesc,attr"`
	PluginName                 *string  `xml:"pluginName,attr"`
	PluginFormats              *string  `xml:"pluginFormats,attr"`
	PluginCharacteristicsValue *string  `xml:"pluginCharacteristicsValue,attr"`
	BinaryDataNamespace        *string  `xml:"binaryDataNamespace,attr"`
	IncludeBinaryInJuceHeader  *int     `xml:"includeBinaryInJuceHeader,attr"`
	AddUsingNamespace          *string  `xml:"addUsingNamespaceToJuceHeader,attr"`
	PluginVST3Category         *string  `xml:"pluginVST3Category,attr"`
	PluginAAXCategory          *string  `xml:"pluginAAXCategory,attr"`
	PluginAUMainType           *string  `xml:"pluginAUMainType,attr"`
	Defines                    *string  `xml:"defines,attr"`
	CompanyEmail               *string  `xml:"companyEmail,attr"`
	CompanyWebsite             *string  `xml:"companyWebsite,attr"`
	CompanyCopyright           *string  `xml:"companyCopyright,attr"`

	HeaderPath *string     `xml:"headerPath,attr"`
	Modules    []RawModule `xml:"MODULES>MODULE"`
	MainGroup  RawGroup    `xml:"MAINGROUP"`
}

type PluginProject struct {
	Name                      string
	Version                   string
	CompanyName               string
	PluginManufacturerCode    string
	PluginManufacturer        string
	PluginCode                string
	PluginDesc                string
	PluginName                string
	PluginFormats             []string
	AAXDisableBypass          string   // "TRUE" or "FALSE"
	AAXDisableMultiMono       string
	EditorRequiresKeys        string
	IsMidiEffect              string
	IsSynth                   string
	WantsMidiInput            string
	ProducesMidiOut           string
	BinaryDataNamespace       string
	IncludeBinaryInJuceHeader int
	AddUsingNamespace         string
	PluginVST3Category        []string
	PluginAAXCategory         []string
	PluginAUMainType          string
	Defines                   []string
	CompanyEmail              string
	CompanyWebsite            string
	CompanyCopyright          string

	Modules     []string
	SourceFiles []string
	AssetFiles  []string
	HeaderDirs  []string
}

func getString(ptr *string, defaultVal string) string {
	if ptr != nil {return *ptr}
	return defaultVal
}

func getInt(ptr *int, defaultVal int) int {
	if ptr != nil {return *ptr}
	return defaultVal
}