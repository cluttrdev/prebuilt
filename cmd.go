package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	// "runtime/trace"

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

	// // Create a file to store the trace output
	// f, err := os.Create("trace.out")
	// if err != nil {
	// 	return fmt.Errorf("failed to create trace output file: %v", err)
	// }
	// defer f.Close()
	//
	// // Start the trace
	// if err := trace.Start(f); err != nil {
	// 	return fmt.Errorf("failed to start trace: %v", err)
	// }
	// defer trace.Stop()

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

	logFile   *os.File
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
	if stateDir, err := userStateDir(); err == nil {
		c.logFile, _ = os.OpenFile(filepath.Join(stateDir, "prebuilt.log"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModePerm)
	}
	if c.logFile == nil {
		c.logFile = os.Stderr
	}

	level := c.logLevel
	if c.debug {
		level = "debug"
	}
	initLogging(c.logFile, level, c.logFormat)
}

func userStateDir() (string, error) {
	xdgStateHome, ok := os.LookupEnv("XDG_STATE_HOME")
	if !ok || xdgStateHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		xdgStateHome = filepath.Join(home, ".local", "state")
	}

	return xdgStateHome, nil
}
