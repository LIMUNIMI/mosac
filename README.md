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

### Warnings
- **AAX** formats is not well supported for Windows due to cross-compilation. Additionalliy on MacOS it can only be used in Pro Tools Developer, if you want to use it in regular Pro Tools you must send a "plugin-signing" email to Avid.
- Please note that all the artefacts compiled by this script are **not signed**.