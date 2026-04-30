package cmd

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fastclaw-ai/weclaw/api"
	"github.com/fastclaw-ai/weclaw/config"
)

type managedProcess struct {
	PID  int
	Args []string
}

func resolveAPIAddr() (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	if apiAddrFlag != "" {
		return apiAddrFlag, nil
	}
	if cfg.APIAddr != "" {
		return cfg.APIAddr, nil
	}
	return api.DefaultAddr, nil
}

func inspectRuntimeState() (pid int, hasPIDFile bool, live []managedProcess, err error) {
	pid, err = readPid()
	if err == nil {
		hasPIDFile = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return 0, false, nil, err
	}

	live, err = findManagedWeclawProcesses()
	if err != nil {
		return 0, hasPIDFile, nil, err
	}
	slices.SortFunc(live, func(a, b managedProcess) int {
		return a.PID - b.PID
	})
	return pid, hasPIDFile, live, nil
}

func stopManagedWeclaw() error {
	_, _, live, err := inspectRuntimeState()
	if err != nil {
		return err
	}

	targets := make([]int, 0, len(live))
	for _, proc := range live {
		targets = append(targets, proc.PID)
	}
	if len(targets) == 0 {
		_ = os.Remove(pidFile())
		return nil
	}

	signalProcesses(targets, syscall.SIGTERM)
	if survivors := waitForProcessesExit(targets, 5*time.Second); len(survivors) > 0 {
		signalProcesses(survivors, syscall.SIGKILL)
		if survivors = waitForProcessesExit(survivors, 2*time.Second); len(survivors) > 0 {
			return fmt.Errorf("managed weclaw processes did not exit: %v", survivors)
		}
	}

	_ = os.Remove(pidFile())
	return nil
}

func waitForAPIReady(pid int, addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processExists(pid) {
			return fmt.Errorf("background process %d exited before API became ready", pid)
		}
		if err := checkAPIHealth(addr); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for API server at %s", addr)
}

func waitForAPIAddrFree(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			_ = ln.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("api address %s is already in use", addr)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func checkAPIHealth(addr string) error {
	client := &http.Client{
		Timeout: 300 * time.Millisecond,
		Transport: &http.Transport{
			Proxy:             nil,
			DisableKeepAlives: true,
		},
	}
	resp, err := client.Get("http://" + addr + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	return nil
}

func findManagedWeclawProcesses() ([]managedProcess, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	self := os.Getpid()
	procs := make([]managedProcess, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid == self {
			continue
		}

		args, err := readProcessArgs(pid)
		if err != nil || !isManagedWeclawProcess(args) {
			continue
		}
		procs = append(procs, managedProcess{PID: pid, Args: args})
	}
	return procs, nil
}

func readProcessArgs(pid int) ([]string, error) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cmdline"))
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(data), "\x00")
	args := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			args = append(args, part)
		}
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("empty cmdline")
	}
	return args, nil
}

func isManagedWeclawProcess(args []string) bool {
	if len(args) == 0 {
		return false
	}
	base := filepath.Base(args[0])
	if !strings.Contains(base, "weclaw") {
		return false
	}

	hasStart := false
	hasForeground := false
	for _, arg := range args[1:] {
		switch arg {
		case "start":
			hasStart = true
		case "-f", "--foreground":
			hasForeground = true
		}
	}
	return hasStart && hasForeground
}

func signalProcesses(pids []int, sig syscall.Signal) {
	for _, pid := range pids {
		if p, err := os.FindProcess(pid); err == nil {
			_ = p.Signal(sig)
		}
	}
}

func waitForProcessesExit(pids []int, timeout time.Duration) []int {
	deadline := time.Now().Add(timeout)
	for {
		survivors := make([]int, 0, len(pids))
		for _, pid := range pids {
			if processExists(pid) {
				survivors = append(survivors, pid)
			}
		}
		if len(survivors) == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return survivors
		}
		time.Sleep(200 * time.Millisecond)
	}
}
