# prebuilt

A tool to manage installations of prebuilt binaries.

## Configuration

```yaml
global:
  installDir: $HOME/.local/bin

binaries: []
  # - name: prebuilt
  #   version: v0.1.0
  #   downloadUrl: https://github.com/cluttrdev/prebuilt/releases/download/{{ .Version }}/prebuilt_{{ .Version }}_linux-amd64.tar.gz
  #   extractPath: prebuilt
```
