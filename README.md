# MOSAC

MOSAC is a Multi-OS Audio Compiler written in **Golang**. Its purpose is being able to compile JUCE/C++ code into various audio plugins format for MacOS, Windows and Linux only by running on an Apple Silicon computer.

Supported formats:

- Standalone
- VST3
- AU (MacOS)
- LV2
- Unity
- AAX (MacOS with JUCE 8 or newer)

## Requirements

#### Using the source repository

- MacOS on Apple Silicon (M1 or newer)
- [Go 1.26.4](https://go.dev/doc/install) or newer
- [Docker Desktop](https://docs.docker.com/desktop/setup/install/mac-install/)
- Xcode Command Line Tools (`xcodebuild`)
- Internet access for downloading JUCE and building Docker images

#### Using a release executable

Go is not required when using a binary downloaded from the [GitHub Releases](https://github.com/LIMUNIMI/mosac/releases) page. The machine still needs MacOS on Apple Silicon, Docker Desktop and Xcode Command Line Tools for MacOS builds.

Download `mosac` and make it executable.
The commands below are the same for a binary built from source and for a release binary. Use `go run path/to/mosac` instead of `./mosac` when running directly from the repository without building an executable.

## Installation

### From the repository

```sh
git clone https://github.com/LIMUNIMI/mosac.git \
cd mosac \
go mod tidy \
go build -o mosac cmd/mosac/main.go
```

Run the first diagnostic check before building a project:

```sh
./mosac health
```

The health check verifies builds the required Linux and Windows Docker images and checks the installed JUCE versions. On MacOS it also checks or builds Projucer for every installed JUCE version.

## Command overview

```text
mosac <command> [options]

create-batch    Generate a batch.csv file for multiple projects
health          Check Docker, JUCE and Projucer state
update          Download or replace a JUCE installation
build           Build one project or all projects in a batch file
```

For command-specific help write `-h` after every subcommand.

## JUCE version management: `update`

`update` downloads a JUCE release from GitHub and installs it under `~/.mosac/juce/<major-version>`. If that major version is already installed, it is replaced.

Install the latest JUCE release:

```sh
./mosac update
```

Install a specific release:

```sh
./mosac update -v 8.0.14
```

MOSAC uses the major-version directories (example) `~/.mosac/juce/7` and `~/.mosac/juce/8` when a project or batch file refers to `JUCE7` or `JUCE8`.

## Diagnostics: `health`

```sh
./mosac health
```

The command checks `xcodebuild` on MacOS, the Docker daemon, the Linux and Windows Docker images, JUCE installations under `~/.mosac/juce`, and Projucer availability for each JUCE version on MacOS. If no JUCE version is installed, use `update` first.

## Creating a batch file: `create-batch`

`create-batch` scans one directory and treats every direct child directory as a JUCE project. It creates `batch.csv` in the directory passed with `-out`.

```sh
./mosac create-batch \
  -dir /home/user/plugins \
  -juce 8 \
  -out /home/user/batches \
  -c Release \
  -sys MacOS,Linux,Windows \
  -formats Standalone,VST3,AU,LV2,Unity,AAX
```

Options:

| Option | Description | Default |
| --- | --- | --- |
| `-dir` | Directory containing project directories | required |
| `-juce` | Default JUCE path or version | required |
| `-out` | Directory where `batch.csv` is written | required |
| `-c` | `Debug` or `Release` | `Release` |
| `-sys` | Comma-separated target systems | `MacOS,Linux,Windows` |
| `-formats` | Comma-separated plugin formats | `Standalone,VST3,AU,LV2,Unity,AAX` |

When a project contains `mosac.conf`, the generated row leaves the JUCE column empty. During the build MOSAC resolves the JUCE version from that project configuration.

## Batch file format

Each non-comment row contains exactly five comma-separated columns:

```text
project_path,juce_path,build_configuration,OS,plugin_formats
```

Systems and formats inside their columns are separated by semicolons:

```csv
# project_path,juce_path,build_configuration,OS,plugin_formats
/home/user/plugins/plugin1,8,Release,MacOS;Linux;Windows,Standalone;VST3;AU;LV2
/home/user/plugins/plugin2,,Debug,MacOS,VST3;AAX
/home/user/plugins/plugin3,7,Release,Linux,Standalone;LV2
```

Column rules:

1. `project_path`: path to the project directory.
2. `juce_path`: an absolute JUCE directory, `7`, `8`, `JUCE7` or `JUCE8`. Leave it empty when the project has `mosac.conf`.
3. `build_configuration`: `Debug` or `Release`.
4. `OS`: one or more of `MacOS`, `Linux` and `Windows`, separated by `;`.
5. `plugin_formats`: one or more supported formats, separated by `;`.

Build the complete batch with:

```sh
./mosac build -b /home/user/batches/batch.csv -OP /home/user/builds
```

`-OP` is required for batch builds. Add `-new` to remove previous build directories before compiling (recommended):

```sh
./mosac build -b /home/user/batches/batch.csv -OP /home/user/builds -new
```

## Project structure

Every project compiled by MOSAC should follow this structure:

```text
ProjectDir/
├── Libraries/             # optional
│   ├── Library1/
│   │   ├── file.cpp
│   │   └── file.h
│   └── Library2/
├── plugin.jucer           # exactly one .jucer file
├── mosac.conf             # optional
├── README.md              # optional
├── Installers/            # optional, copied to the output
├── Presets/               # optional, copied to the output
├── pluginName.png         # optional, copied to the output
└── Source/
    ├── pluginCode.cpp
    └── pluginCode.h
```

External libraries must be inside `Libraries/` and referenced by the `.jucer` project through Projucer. `Presets/` and `Installers/`, when present, are copied to the output directory.

## Project configuration: `mosac.conf`

`mosac.conf` is optional. When present, it supplies the JUCE version and metadata used in the build output. It must contain four or five lines in this order:

```text
JUCE-8
english plugin description
Author1,Author2
author1@example.com,author2@example.com
https://example.com
```

The fifth line, the URL, is optional. The first line accepts `JUCEn` or `JUCE-n` (*n* is the major JUCE version).

Example:

```text
JUCE-7
A synthesizer plugin for research and live performance.
Carlo Ancri,Autor Two
carlo@gmail.com,author2@gmail.com
https://example.com/plugin
```

When `mosac.conf` exists, a single-project build can omit `-JP`:

```sh
./mosac build \
  -PP /home/user/plugins/plugin1 \
  -OP /home/user/builds \
  -sys MacOS \
  -formats Standalone,VST3
```

Without `mosac.conf`, provide the JUCE path explicitly with `-JP`:

```sh
./mosac build \
  -PP /home/user/plugins/plugin1 \
  -JP 8 \
  -OP /home/user/builds \
  -c Release \
  -sys MacOS,Linux,Windows \
  -formats Standalone,VST3,AU,LV2,Unity
```

## Building a single project: `build`

The single-project mode requires `-PP` and `-OP`. Use `-JP` unless the project contains a valid `mosac.conf`.

```sh
./mosac build \
  -PP /home/user/plugins/MyPlugin \
  -JP 8 \
  -OP /home/user/builds \
  -c Release \
  -sys MacOS \
  -formats Standalone,VST3,AU
```

Options:

| Option | Description | Default |
| --- | --- | --- |
| `-PP` | Single project path | empty |
| `-JP` | JUCE path or version | empty when `mosac.conf` is present |
| `-OP` | Output directory | required |
| `-c` | `Debug` or `Release` | `Release` |
| `-sys` | Comma-separated target systems | `MacOS,Linux,Windows` |
| `-formats` | Comma-separated plugin formats | `Standalone,VST3,AU,LV2,Unity,AAX` |
| `-new` | Remove previous build directories before compiling | `false` |

Examples for individual target systems:

```sh
# Linux plugins
./mosac build -PP /home/user/plugins/MyPlugin -JP 8 -OP /home/user/builds -sys Linux -formats Standalone,VST3,LV2

# Windows plugins through Docker cross-compilation
./mosac build -PP /home/user/plugins/MyPlugin -JP 8 -OP /home/user/builds -sys Windows -formats Standalone,VST3,Unity

# MacOS plugins, including AAX with JUCE 8+
./mosac build -PP /home/user/plugins/MyPlugin -JP 8 -OP /home/user/builds -sys MacOS -formats Standalone,VST3,AU,AAX
```

## AAX rules

- On MacOS, AAX can be generated only with JUCE 8 or newer. With JUCE 7, MOSAC removes AAX from the generated `.jucer` and `CMakeLists.txt` formats.
- AAX is never included in the CMake build used for Linux or Windows.
- A request containing only `AAX` with JUCE 7 is rejected because no buildable format remains.
- AAX output is not signed. It can be used in Pro Tools Developer; regular Pro Tools requires the Avid plugin-signing process.

## Notes

- Build artefacts are not signed.
- File names and `#include` paths must use consistent casing: Linux is case-sensitive.
- MacOS builds use the Projucer/Xcode project generated by the selected JUCE version. Linux and Windows builds use the generated CMake project and Docker.
- Only one MOSAC process can run at a time. A stale lock file at `~/.mosac/mosac.lock` may be removed manually only when MOSAC is not running.
