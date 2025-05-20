package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
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

func initLogging(w io.Writer, level string, format string) {
	if w == nil {
		w = os.Stderr
	}

	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := slog.HandlerOptions{
		Level: lvl,
	}

	var handler slog.Handler
	switch format {
	case "text":
		handler = slog.NewTextHandler(w, &opts)
	case "json":
		handler = slog.NewJSONHandler(w, &opts)
	default:
		handler = slog.NewTextHandler(w, &opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

type rootCmd struct {
	ConfigFile string

	logLevel  string
	logFormat string
	debug     bool
}

func (c *rootCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.ConfigFile, "config", ".prebuilt.yaml", "The configuration file.")

	fs.StringVar(&c.logLevel, "log-level", "info", "The log level.")
	fs.StringVar(&c.logFormat, "log-format", "text", "The log format ('text' or 'json').")
	fs.BoolVar(&c.debug, "debug", false, "Enable debug mode.")
}

func (c *rootCmd) Exec(ctx context.Context, args []string) error {
	return flag.ErrHelp
}

func (c *rootCmd) initLogging() {
	level := c.logLevel
	if c.debug {
		level = "debug"
	}
	initLogging(os.Stderr, level, c.logFormat)
}
