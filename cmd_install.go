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
	"sync"

	"github.com/cluttrdev/cli"
	"github.com/pterm/pterm"
	"go.cluttr.dev/prebuilt/internal/metaerr"
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

func (c *installCmd) Exec(ctx context.Context, args []string) (err error) {
	c.initLogging()

	defer func() {
		if err != nil && c.rootCmd.logFile != os.Stderr {
			err = fmt.Errorf("%w\nSee %s for details.", err, c.rootCmd.logFile.Name())
		}
	}()

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

	tmpDir, err := os.MkdirTemp("", "prebuilt-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			slog.Error("failed to remove temporary directory", "dir", tmpDir, "error", err)
		}
	}()

	installDir := expandPath(cfg.Global.InstallDir)

	// set up fancy output
	var (
		multiPrinter = pterm.DefaultMultiPrinter
		failedSpecs  []BinarySpec
		wg sync.WaitGroup
	)
	_, _ = multiPrinter.Start()
	for _, spec := range binaries {
		wg.Add(1)
		go func() {
			defer wg.Done()
			spinner, _ := pterm.DefaultSpinner.WithWriter(multiPrinter.NewWriter()).Start("Installing ", spec.Name)
			if err := c.processBinary(spec, tmpDir, installDir); err != nil {
				slog.With("name", spec.Name, "error", err).
					With(metaerr.GetMetadata(err)...).
					Error("failed to install binary")
				failedSpecs = append(failedSpecs, spec)
				spinner.Fail("Failed to install ", spec.Name, ": ", err)
				return
			}
			spinner.Success()
		}()
	}
	wg.Wait()
	_, _ = multiPrinter.Stop()
	if len(failedSpecs) > 0 {
		names := make([]string, 0, len(failedSpecs))
		for _, spec := range failedSpecs {
			names = append(names, spec.Name)
		}
		return fmt.Errorf("installation failed: %v", names)
	}

	return nil
}

func (c *installCmd) processBinary(spec BinarySpec, tmpDir string, installDIr string) error {
	bin, err := resolveBinarySpec(spec)
	if err != nil {
		return fmt.Errorf("resolve binary spec: %w", err)
	}

	path, err := Download(bin.DownloadURL, tmpDir)
	if err != nil {
		return fmt.Errorf("download binary asset: %w", err)
	}

	if bin.ExtractPath != "" {
		path, err = Extract(path, bin.ExtractPath)
		if err != nil {
			return fmt.Errorf("extract archived binary: %w", err)
		}
	}

	out := filepath.Join(installDIr, bin.Name)
	if err := Install(path, out); err != nil {
		return fmt.Errorf("install binary: %w", err)
	}

	return nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		path = filepath.Join("${HOME}", path[1:])
	}
	return os.ExpandEnv(path)
}
