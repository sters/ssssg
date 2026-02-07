package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/spf13/cobra"
	"github.com/sters/ssssg/ssssg"
)

//nolint:gochecknoglobals
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func getVersion() string {
	if version != "dev" {
		return version
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}

	return version
}

func getCommit() string {
	if commit != "none" {
		return commit
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				if len(setting.Value) > 7 {
					return setting.Value[:7]
				}

				return setting.Value
			}
		}
	}

	return commit
}

func getDate() string {
	if date != "unknown" {
		return date
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.time" {
				if t, err := time.Parse(time.RFC3339, setting.Value); err == nil {
					return t.UTC().Format("2006-01-02T15:04:05Z")
				}

				return setting.Value
			}
		}
	}

	return date
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "ssssg",
		Short: "Super Simple Static Site Generator",
	}

	buildCmd := newBuildCmd()
	initCmd := newInitCmd()
	versionCmd := newVersionCmd()

	rootCmd.AddCommand(buildCmd, initCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newBuildCmd() *cobra.Command {
	var (
		configPath  string
		templateDir string
		staticDir   string
		outputDir   string
		timeout     time.Duration
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the static site",
		RunE: func(_ *cobra.Command, _ []string) error {
			return ssssg.Build(context.Background(), ssssg.BuildOptions{
				ConfigPath:  configPath,
				TemplateDir: templateDir,
				StaticDir:   staticDir,
				OutputDir:   outputDir,
				Timeout:     timeout,
				Log:         os.Stdout,
			})
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "site.yaml", "path to config file")
	cmd.Flags().StringVar(&templateDir, "templates", "", "path to templates directory")
	cmd.Flags().StringVar(&staticDir, "static", "", "path to static directory")
	cmd.Flags().StringVar(&outputDir, "output", "", "path to output directory")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "timeout for HTTP fetches")

	return cmd
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a new site project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			if err := ssssg.Init(dir); err != nil {
				return fmt.Errorf("init: %w", err)
			}

			fmt.Printf("Initialized new ssssg project in %s\n", dir)

			return nil
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("Version:    %s\n", getVersion())
			fmt.Printf("Commit:     %s\n", getCommit())
			fmt.Printf("Built:      %s\n", getDate())
			fmt.Printf("Go version: %s\n", runtime.Version())
			fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
}
