package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// Version is the public release version and is set at build time via -ldflags.
var Version = "dev"

// PublicCommit is the commit used for the public release artifact.
var PublicCommit = "unknown"

// InternalVersion is the internal development/pre-release marker used for handoff and issue triage.
var InternalVersion = "dev"

// InternalCommit is the commit corresponding to InternalVersion.
var InternalCommit = "unknown"

var rootCmd = &cobra.Command{
	Use:     "weclaw",
	Short:   "WeChat AI agent bridge",
	Long:    "weclaw bridges WeChat messages to AI agents via the iLink API.",
	Version: publicVersion(),
	RunE:    runStart, // default command is start
}

func init() {
	rootCmd.SetVersionTemplate(versionOutput(runtime.GOOS, runtime.GOARCH) + "\n")
}

func publicVersion() string {
	return cleanVersionField(Version, "dev")
}

func cleanVersionField(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func shortCommit(value string) string {
	value = cleanVersionField(value, "unknown")
	if value == "unknown" {
		return value
	}
	if len(value) > 7 {
		return value[:7]
	}
	return value
}

func versionOutput(goos, goarch string) string {
	platform := goos + "/" + goarch
	return fmt.Sprintf(
		"weclaw public version: %s | %s (%s)\nweclaw internal version: %s | %s (%s)",
		publicVersion(),
		shortCommit(PublicCommit),
		platform,
		cleanVersionField(InternalVersion, "dev"),
		shortCommit(InternalCommit),
		platform,
	)
}

var releaseLDFlags = []string{
	"-X github.com/fastclaw-ai/weclaw/cmd.Version=<public-release-tag>",
	"-X github.com/fastclaw-ai/weclaw/cmd.PublicCommit=<public-release-commit>",
	"-X github.com/fastclaw-ai/weclaw/cmd.InternalVersion=<internal-p-tag>",
	"-X github.com/fastclaw-ai/weclaw/cmd.InternalCommit=<internal-p-commit>",
}

func releaseMetadataLDFlags() []string {
	return append([]string(nil), releaseLDFlags...)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
