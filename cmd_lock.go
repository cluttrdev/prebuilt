package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/cluttrdev/cli"
	"github.com/goccy/go-yaml"
	"github.com/pterm/pterm"

	"go.cluttr.dev/prebuilt/internal/metaerr"
)

func newLockCmd() *cli.Command {
	cfg := lockCommand{}

	fs := flag.NewFlagSet("prebuilt lock", flag.ExitOnError)

	cfg.RegisterFlags(fs)

	return &cli.Command{
		Name:       "lock",
		ShortHelp:  "Update the lockfile.",
		ShortUsage: "prebuilt lock [OPTION]...",
		Flags:      fs,
		Exec:       cfg.Exec,
	}
}

type lockCommand struct {
	rootCmd

	resolver Resolver
}

func (c *lockCommand) RegisterFlags(fs *flag.FlagSet) {
	c.rootCmd.RegisterFlags(fs)
}

func (c *lockCommand) Exec(ctx context.Context, args []string) (err error) {
	c.initLogging()

	defer func() {
		if err != nil && c.logFile != os.Stderr {
			err = fmt.Errorf("%w\nSee %s for details", err, c.logFile.Name())
		}
	}()

	var cfg Config
	if err := LoadConfigFile(c.ConfigFile, &cfg); err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	if err := c.resolver.Init(cfg.Providers); err != nil {
		return fmt.Errorf("initialize providers: %w", err)
	}

	spinner, _ := pterm.DefaultSpinner.Start("Resolving binaries")
	lock, err := c.resolver.Resolve(ctx, cfg.Binaries)
	if err != nil {
		slog.With("error", err).
			With(metaerr.GetMetadata(err)...).
			Error("failed to resolve binaries")
		spinner.Fail()
		return err
	}
	spinner.Success()

	lockfile := replaceFileExt(c.ConfigFile, ".lock")
	return writeLockFile(lockfile, lock)
}

func readLockFile(name string) (Lock, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return Lock{}, err
	}
	var lock Lock
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return Lock{}, err
	}
	return lock, nil
}

func writeLockFile(name string, lock Lock) error {
	data, err := yaml.Marshal(lock)
	if err != nil {
		return err
	}
	return os.WriteFile(name, data, 0644)
}

func replaceFileExt(path string, ext string) string {
	oldExt := filepath.Ext(path)
	return path[:len(path)-len(oldExt)] + ext
}
