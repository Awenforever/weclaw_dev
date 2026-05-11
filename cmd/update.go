package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

const githubRepo = "Awenforever/weclaw_dev"

var uninstallPurgeFlag bool

const updateCheckInterval = 24 * time.Hour

func init() {
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(versionCmd)

	uninstallCmd.Flags().BoolVar(&uninstallPurgeFlag, "purge", false, "also remove ~/.weclaw user data")
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("weclaw %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update weclaw to the latest GitHub release",
	RunE:  runUpdate,
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade weclaw to the latest GitHub release",
	RunE:  runUpdate,
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the installed weclaw binary",
	RunE:  runUninstall,
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

type updateCheckState struct {
	LastCheckedAt string `json:"last_checked_at,omitempty"`
	LatestVersion string `json:"latest_version,omitempty"`
	LastNotified  string `json:"last_notified,omitempty"`
}

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Println("Checking for updates...")

	ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
	defer cancel()

	latest, err := getLatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to check latest version: %w", err)
	}

	if latest == Version {
		fmt.Printf("Already up to date (%s)\n", Version)
		return nil
	}

	fmt.Printf("Current: %s -> Latest: %s\n", Version, latest)

	filename := releaseAssetName(runtime.GOOS, runtime.GOARCH)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", githubRepo, latest, filename)

	fmt.Printf("Downloading %s...\n", url)
	tmpFile, err := downloadFile(ctx, url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpFile)

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}
	if resolved, err := resolveSymlink(exePath); err == nil {
		exePath = resolved
	}

	if err := replaceBinary(tmpFile, exePath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}

	clearMacQuarantineAttrs(exePath)

	fmt.Printf("Updated to %s\n", latest)

	_, _, live, err := inspectRuntimeState()
	if err != nil {
		return err
	}
	targets := managedProcessPIDsForExecutable(live, exePath)
	if len(targets) > 0 {
		fmt.Println("Stopping old process...")
		if err := stopManagedPIDs(targets); err != nil {
			return err
		}
		_ = os.Remove(pidFile())

		apiAddr, err := resolveAPIAddr()
		if err != nil {
			return fmt.Errorf("failed to resolve API address: %w", err)
		}

		fmt.Println("Starting new version...")
		if err := runDaemon(false, apiAddr, "", ""); err != nil {
			fmt.Printf("Update complete, but restart failed: %v\n", err)
			fmt.Println("Please run 'weclaw start' manually.")
		}
	} else {
		fmt.Println("Update complete. Run 'weclaw start' to start.")
	}

	return nil
}

func runUninstall(cmd *cobra.Command, args []string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("self-uninstall is not supported on Windows; remove the weclaw executable manually")
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}
	if resolved, err := resolveSymlink(exePath); err == nil {
		exePath = resolved
	}

	_, _, live, err := inspectRuntimeState()
	if err != nil {
		return err
	}
	targets := managedProcessPIDsForExecutable(live, exePath)
	if len(targets) > 0 {
		fmt.Println("Stopping managed weclaw process for this executable...")
		if err := stopManagedPIDs(targets); err != nil {
			return err
		}
		_ = os.Remove(pidFile())
	}

	fmt.Printf("Removing %s...\n", exePath)
	if err := removeBinary(exePath); err != nil {
		return fmt.Errorf("remove binary: %w", err)
	}

	if uninstallPurgeFlag {
		dataDir := weclawDir()
		fmt.Printf("Removing user data at %s...\n", dataDir)
		if err := os.RemoveAll(dataDir); err != nil {
			return fmt.Errorf("remove user data: %w", err)
		}
		fmt.Println("weclaw uninstalled and user data removed.")
		return nil
	}

	fmt.Println("weclaw binary removed.")
	fmt.Printf("User data is preserved at %s\n", weclawDir())
	return nil
}

func getLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "weclaw-update-checker")

	resp, err := updateHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	if release.TagName == "" {
		return "", fmt.Errorf("latest release has no tag")
	}
	return release.TagName, nil
}

func downloadFile(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "weclaw-upgrader")

	resp, err := updateHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "weclaw-update-*")
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}

	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}

	return tmp.Name(), nil
}

func updateHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func releaseAssetName(goos, goarch string) string {
	name := fmt.Sprintf("weclaw_%s_%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

func replaceBinary(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	if runtime.GOOS != "windows" {
		fmt.Printf("Installing to %s (requires sudo)...\n", dst)
		cmd := exec.Command("sudo", "cp", src, dst)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return fmt.Errorf("cannot write to %s", dst)
}

func removeBinary(path string) error {
	if err := os.Remove(path); err == nil {
		return nil
	}

	if runtime.GOOS != "windows" {
		fmt.Printf("Removing %s requires sudo...\n", path)
		cmd := exec.Command("sudo", "rm", "-f", path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return fmt.Errorf("cannot remove %s", path)
}

func stopManagedPIDs(targets []int) error {
	if len(targets) == 0 {
		return nil
	}

	signalProcesses(targets, syscall.SIGTERM)
	if survivors := waitForProcessesExit(targets, 5*time.Second); len(survivors) > 0 {
		signalProcesses(survivors, syscall.SIGKILL)
		if survivors = waitForProcessesExit(survivors, 2*time.Second); len(survivors) > 0 {
			return fmt.Errorf("managed weclaw processes did not exit: %v", survivors)
		}
	}
	return nil
}

func managedProcessPIDsForExecutable(live []managedProcess, exePath string) []int {
	var targets []int
	for _, proc := range live {
		if len(proc.Args) == 0 {
			continue
		}
		if sameExecutablePath(proc.Args[0], exePath) {
			targets = append(targets, proc.PID)
		}
	}
	return targets
}

func sameExecutablePath(candidate, expected string) bool {
	candidatePath, candidateErr := normalizeExecutablePath(candidate)
	expectedPath, expectedErr := normalizeExecutablePath(expected)
	if candidateErr != nil || expectedErr != nil {
		return false
	}

	candidateInfo, candidateStatErr := os.Stat(candidatePath)
	expectedInfo, expectedStatErr := os.Stat(expectedPath)
	if candidateStatErr == nil && expectedStatErr == nil {
		return os.SameFile(candidateInfo, expectedInfo)
	}

	return candidatePath == expectedPath
}

func normalizeExecutablePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty executable path")
	}
	if !filepath.IsAbs(path) {
		resolved, err := exec.LookPath(path)
		if err != nil {
			return "", err
		}
		path = resolved
	}
	if resolved, err := resolveSymlink(path); err == nil {
		path = resolved
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func resolveSymlink(path string) (string, error) {
	for {
		target, err := os.Readlink(path)
		if err != nil {
			return path, nil
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		path = target
	}
}

func clearMacQuarantineAttrs(path string) {
	if runtime.GOOS != "darwin" {
		return
	}
	exec.Command("xattr", "-d", "com.apple.quarantine", path).Run()
	exec.Command("xattr", "-d", "com.apple.provenance", path).Run()
}

func maybePrintUpdateNotice(w io.Writer) {
	if w == nil || Version == "" || Version == "dev" {
		return
	}

	now := time.Now()
	state, _ := readUpdateCheckState()
	if !updateCheckDue(state, now) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	latest, err := getLatestVersion(ctx)
	if err != nil {
		return
	}

	state.LastCheckedAt = now.Format(time.RFC3339)
	state.LatestVersion = latest

	if shouldOfferUpdate(Version, latest) && state.LastNotified != latest {
		fmt.Fprintf(w, "Update available: weclaw %s -> %s. Run: weclaw upgrade\n", Version, latest)
		state.LastNotified = latest
	}

	_ = writeUpdateCheckState(state)
}

func shouldOfferUpdate(current, latest string) bool {
	current = strings.TrimSpace(current)
	latest = strings.TrimSpace(latest)
	return current != "" && latest != "" && current != latest && current != "dev"
}

func updateCheckDue(state updateCheckState, now time.Time) bool {
	if state.LastCheckedAt == "" {
		return true
	}
	last, err := time.Parse(time.RFC3339, state.LastCheckedAt)
	if err != nil {
		return true
	}
	return now.Sub(last) >= updateCheckInterval
}

func updateCheckStatePath() string {
	return filepath.Join(weclawDir(), "update-check.json")
}

func readUpdateCheckState() (updateCheckState, error) {
	path := updateCheckStatePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return updateCheckState{}, err
	}
	var state updateCheckState
	if err := json.Unmarshal(data, &state); err != nil {
		return updateCheckState{}, err
	}
	return state, nil
}

func writeUpdateCheckState(state updateCheckState) error {
	path := updateCheckStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}
