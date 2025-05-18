package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cluttrdev/cli"
)

func newInstallCmd() *cli.Command {
	var cfg installCmd

	fs := flag.NewFlagSet("prebuilt install", flag.ExitOnError)

	cfg.RegisterFlags(fs)

	return &cli.Command{
		Name:       "install",
		ShortHelp:  "Install prebuilt binaries.",
		ShortUsage: "prebuilt install [OPTION]... [NAME]...",
		Flags:      fs,
		Exec:       cfg.Exec,
	}
}

type installCmd struct {
	rootCmd
}

func (c *installCmd) RegisterFlags(fs *flag.FlagSet) {
	c.rootCmd.RegisterFlags(fs)
}

func (c *installCmd) Exec(ctx context.Context, args []string) error {
	var cfg Config
	if err := LoadConfigFile(c.ConfigFile, &cfg); err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	var binaries []BinarySpec
	if len(args) > 0 {
		for _, name := range args {
			index := slices.IndexFunc(cfg.Binaries, func(b BinarySpec) bool {
				return b.Name == name
			})
			if index == -1 {
				return fmt.Errorf("name %s not found", name)
			}
			binaries = append(binaries, cfg.Binaries[index])
		}
	} else {
		binaries = cfg.Binaries
	}

	dir, err := os.MkdirTemp("", "prebuilt-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			slog.Error("failed to remove temporary directory", "dir", dir, "error", err)
		}
	}()

	for _, spec := range binaries {
		bin, err := resolveBinarySpec(spec)
		if err != nil {
			return err
		}

		fmt.Printf("downloading %s ...\n", bin.DownloadURL)
		path, err := Download(bin.DownloadURL, dir)
		if err != nil {
			return err
		}

		if bin.ExtractPath != "" {
			path, err = Extract(path, bin.ExtractPath)
			if err != nil {
				return err
			}
		}

		out := filepath.Join(expandPath(cfg.Global.InstallDir), bin.Name)
		if err := Install(path, out); err != nil {
			return err
		}
	}

	return nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		path = filepath.Join("${HOME}", path[1:])
	}
	return os.ExpandEnv(path)
}
