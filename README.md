# prebuilt

A tool to manage installations of prebuilt binaries.

## Configuration

```yaml
global:
  installDir: $HOME/.local/bin

binaries: []
  # - name: prebuilt
  #   version: latest
  #   provider: github://cluttrdev/prebuilt?asset=prebuilt_{{ .Version }}_linux_amd64.tar.gz
```
