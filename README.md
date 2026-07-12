# MOSAC
MOSAC is a Multi-OS Audio Compiler written in **Golang**.
Its purpose is being able to compile JUCE/C++ code into various audio plugins format (Standalone, VST3, AU, LV2, Unity, AAX) for MacOS, Windows and Linux only by running on an Apple M1 (or newer) computer.

---

### Requirements
- Docker
- Golang (v1.26.4 or newer)
- CMake (v3.22 or newer)
- JUCE framework
- Xcode (CLI version)

## Interfaces
MOSAC works on **CLI**. The execution requires specific parameters as detailed in the application's **--help (or -h) section**. Once all parameters have been correctly supplied, the plugin build process is initiated automatically.

*Tip*: before ever running MOSAC for its purpose, make sure to run it first with the `-img` flag in order to prepare all the Docker images needed.

### Batch (multiple builds)
**Batch mode** is activated via the `-batch` parameter, followed by the path to a **CSV file** containing the comprehensive list of project paths to be compiled, the desired JUCE path (specified for compatibility), their respective build configurations, the targeted operating systems and plugins format.
It's recommended to specify also the `-OP` (*outputPath*) parameter.

#### BatchMaker
Inside the `BatchMaker` folder, you will find `BatchMaker.go` which allows you to generate a `batch.csv` file to use with `mosac.go`. This utility accepts **three command-line paths as arguments**: the directory containing the JUCE projects to be compiled with mosac, the JUCE framework path, and the output directory for the batch.csv file.
The **command** to run this script is:
```
go run BatchMaker.go <dirPath> <JucePath> <outputPath>
```
> [!TIP]
This utility is designed for rapid batch file creation. For complex batch requirements (such as using multiple JUCE versions), manual editing is advised.
##### Batch configuration file:

| project_path            | juce_path    |  build_configuration | OS                  | plugin_formats  |
| ----------------------- | ------------ | -------------------- | ------------------- | --------------- |
| /home/user/TEST/plugin1 | path/to/JUCE | Release              | Linux;MacOS;Windows | Standalone;VST3 |
| /home/user/TEST/plugin2 | path/to/JUCE | Debug                | Linux;Windows       | AU;VST3         |
| /home/user/TEST/plugin3 | path/to/JUCE | Release              | Linux               | Standalone      |

``` csv
-------------------
| file: batch.csv |
-------------------
#
# project_path = Path to the project to compile
# juce_path = Path to the JUCE directory
# build_configuration = 'Debug' or 'Release'
# OS = OSs to target for compiling. They must be separated by ';'.
# plugin_formats = List of plugin formats to compile. They must be separated by ';'.
#
/home/PLUGINS/plugin1,/path/to/JUCE/,Release,Linux;MacOS;Windows,Standalone;VST3
/home/PLUGINS/plugin2,/path/to/JUCE/,Debug,Linux,AU;VST3
/home/PLUGINS/plugin3,/path/to/JUCE/,Release,MacOS,Standalone
```

## Installation & first run
1) Download [Golang](https://go.dev/doc/install) v1.26.4 and [Docker](https://docs.docker.com/desktop/setup/install/mac-install/).
2) Clone the JUCE framework directory:
```
git clone https://github.com/juce-framework/JUCE.git
```
3) Clone the repo:
```
git clone https://github.com/Carlo-Unimi/mosac_go.git
```
4) Enter `mosac_go` and run:
```
go mod tidy
```
5) Once the Go project is set run:
```
go build mosac.go
./mosac -img
```
6) Compile a JUCE plugin:
```
./mosac -PP /path/to/JUCEProject -JP /path/to/JUCE -OP /path/to/outputDir -sys Linux,Windows,MacOS -formats Standalone,VST3,AU,Unity,LV2 -b Release
```

## Output
After building the plugin, a `pluginName.json` gets generated as a summary.
```
outputDir
├── plugin1
│   ├── Linux
│   │   ├── LV2
│   │   ├── Standalone
│   │   ├── Unity
│   │   └── VST3
│   ├── MacOS
│   │   ├── AAX
│   │   ├── AU
│   │   ├── LV2
│   │   ├── Standalone
│   │   ├── Unity
│   │   └── VST3
│   └── Windows
│       ├── AAX (warning)
│       ├── LV2
│       ├── Standalone
│       ├── Unity
│       └── VST3
├── plugin1.json
├── plugin2
│   ├── Linux
│   │   ├── LV2
│   │   ├── Standalone
│   │   ├── Unity
│   │   └── VST3
│   ├── MacOS
│   │   ├── AAX
│   │   ├── AU
│   │   ├── LV2
│   │   ├── Standalone
│   │   ├── Unity
│   │   └── VST3
│   └── Windows
│       ├── AAX (warning)
│       ├── LV2
│       ├── Standalone
│       ├── Unity
│       └── VST3
└── plugin2.json
```

### Notes
- Please note that all the artefacts compiled by this script are **not signed**.
- **AAX** formats is not well supported for Windows due to cross-compilation. Additionalliy on MacOS it can only be used in Pro Tools Developer, if you want to use it in regular Pro Tools you must send a "plugin-signing" email to Avid.
- Please ensure that all file names and `#include` paths strictly follow **case-sensitive** naming conventions. While Windows is case-insensitive, the Linux environment is not. Correct casing is essential to avoid compilation errors during cross-platform builds.