package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop all detected managed weclaw processes",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := stopManagedWeclawWithResult()
		if err != nil {
			return err
		}

		if len(result.TargetPIDs) == 0 {
			if result.HadPIDFile {
				fmt.Println("weclaw was not running; cleared stale pid file")
			} else {
				fmt.Println("weclaw was not running")
			}
			fmt.Println("Verified: no managed weclaw process remains")
			return nil
		}

		fmt.Println("weclaw stopped")
		fmt.Printf("Stopped PIDs: %s\n", formatPIDs(result.TargetPIDs))
		if result.PIDFileTargeted {
			fmt.Printf("PID file process was also stopped: %d\n", result.PIDFilePID)
		}
		fmt.Println("Verified: no managed weclaw process remains")
		return nil
	},
}
