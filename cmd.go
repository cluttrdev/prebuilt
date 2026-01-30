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
	"strings"

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
			newLockCmd(),
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
	var (
		stateDir = xdgDir(xdgDataHome)
		err      error
	)
	c.logFile, err = os.OpenFile(filepath.Join(stateDir, "prebuilt.log"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		c.logFile = os.Stderr
	}

	level := c.logLevel
	if c.debug {
		level = "debug"
	}
	initLogging(c.logFile, level, c.logFormat)
}

// loadConfig loads the configuration file.
func (c *rootCmd) loadConfig() (Config, error) {
	path := c.ConfigFile
	if path == "" {
		configDir := xdgDir(xdgConfigHome)
		path = filepath.Join(configDir, "config.yaml")
	}

	var cfg Config
	if err := LoadConfigFile(c.ConfigFile, &cfg); err != nil {
		return Config{}, fmt.Errorf("load configuration: %w", err)
	}

	return cfg, nil
}

// xdgHomeKind represents the kind of XDG home directory.
type xdgHomeKind string

const (
	xdgDataHome   xdgHomeKind = "DATA"
	xdgConfigHome xdgHomeKind = "CONFIG"
	xdgCacheHome  xdgHomeKind = "CACHE"
	xdgStateHome  xdgHomeKind = "STATE"
)

// xdgDir returns the application's directory for the given XDG kind.
func xdgDir(kind xdgHomeKind) string {
	const appName = "prebuilt"

	if path := os.Getenv("PREBUILT_" + string(kind) + "_HOME"); path != "" {
		return filepath.Join(path, appName)
	}

	if path := os.Getenv("PREBUILT_HOME"); path != "" {
		return filepath.Join(path, strings.ToLower(string(kind)), appName)
	}

	if path := os.Getenv("XDG_" + string(kind) + "_HOME"); path != "" {
		return filepath.Join(path, appName)
	}

	if path, _ := os.UserHomeDir(); path != "" {
		switch kind {
		case xdgDataHome:
			return filepath.Join(path, ".local", "share", appName)
		case xdgStateHome:
			return filepath.Join(path, ".local", "state", appName)
		default:
			return filepath.Join(path, strings.ToLower(string(kind)), appName)
		}
	}

	if path, err := os.Getwd(); path != "" && err == nil {
		return filepath.Join(path, strings.ToLower(string(kind)), appName)
	}

	return filepath.Join("."+appName, strings.ToLower(string(kind)))
}
