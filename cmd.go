package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/cluttrdev/cli"
)

// execute configures the root command and then runs it with the given context.
func execute(ctx context.Context) error {
	cmd := configure()
	opts := []cli.ParseOption{
		cli.WithEnvVarPrefix("PREBUILT"),
	}
	args := os.Args[1:]

	if err := cmd.Parse(args, opts...); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("parse arguments: %w", err)
	}

	return cmd.Run(ctx)
}

// configure returns the root command.
func configure() *cli.Command {
	var cfg rootCmd

	fs := flag.NewFlagSet("prebuilt", flag.ExitOnError)

	cfg.RegisterFlags(fs)

	return &cli.Command{
		Name:       "prebuilt",
		ShortHelp:  "Manage prebuilt binaries installations.",
		ShortUsage: "prebuilt [COMMAND] [OPTION]... [ARG]...",
		Subcommands: []*cli.Command{
			cli.DefaultVersionCommand(os.Stdout),
			newInstallCmd(),
		},
		Flags: fs,
		Exec:  cfg.Exec,
	}
}

type rootCmd struct {
	ConfigFile string
}

func (c *rootCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.ConfigFile, "config", ".prebuilt.yaml", "The configuration file.")
}

func (c *rootCmd) Exec(ctx context.Context, args []string) error {
	return flag.ErrHelp
}
