# prebuilt

A tool to manage installations of prebuilt binaries.

## Installation

The first time you install `prebuilt`, you can do it the way you'd usually install
pre-built binaries, e.g.
```sh
# Determine the latest release version
VERSION=$(curl -sSfL https://api.github.com/repos/cluttrdev/prebuilt/releases/latest | jq -r '.tag_name')
# Download the release asset that matches your system
OS=$(uname -s)    # e.g. Linux
ARCH=$(uname -m)  # e.g. x86_64
curl -sSfL -O https://github.com/cluttrdev/prebuilt/releases/download/${VERSION}/prebuilt_${VERSION#v}_${OS}_${ARCH}.tar.gz
# Extract the binary
tar -ozxf ./prebuilt_${VERSION#v}_${OS}_${ARCH}.tar.gz prebuilt
# Install it to a directory that's in your search $PATH
install --mode 0751 ./prebuilt ~/.local/bin/
```

The next time you want to install an update, you can let `prebuilt` do these
steps for you in a single command:
```sh
prebuilt install --update prebuilt
```
if you have a `.prebuilt.yaml` configuration file like below.

## Configuration

```yaml
global:
  installDir: $HOME/.local/bin

binaries:
  - name: prebuilt
    version: latest  # the default
    provider: github://cluttrdev/prebuilt?asset=prebuilt_{{ .Version | trimPrefix "v" }}_Linux_x86_64.tar.gz
    extractPath: prebuilt
```
