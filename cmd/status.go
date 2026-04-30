package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if weclaw is running in background",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, hasPIDFile, live, err := inspectRuntimeState()
		if err != nil {
			return err
		}

		switch len(live) {
		case 0:
			switch {
			case hasPIDFile && processExists(pid):
				fmt.Printf("weclaw is unhealthy: pid file points to live pid=%d, but no managed foreground process was found\n", pid)
			case hasPIDFile:
				fmt.Println("weclaw is not running (stale pid file)")
			default:
				fmt.Println("weclaw is not running")
			}
		case 1:
			livePID := live[0].PID
			if hasPIDFile && pid == livePID {
				fmt.Printf("weclaw is running (pid=%d)\n", livePID)
			} else {
				fmt.Printf("weclaw is unhealthy: live managed pid=%d", livePID)
				if hasPIDFile {
					fmt.Printf(", pid file=%d", pid)
				} else {
					fmt.Print(", pid file missing")
				}
				fmt.Println()
			}
			fmt.Printf("Log: %s\n", logFile())
		default:
			pids := make([]string, 0, len(live))
			for _, proc := range live {
				pids = append(pids, fmt.Sprintf("%d", proc.PID))
			}
			fmt.Printf("weclaw is unhealthy: multiple managed processes detected (pids=%s)\n", strings.Join(pids, ", "))
			if hasPIDFile {
				fmt.Printf("PID file: %d\n", pid)
			} else {
				fmt.Println("PID file: missing")
			}
			fmt.Printf("Log: %s\n", logFile())
		}
		return nil
	},
}
