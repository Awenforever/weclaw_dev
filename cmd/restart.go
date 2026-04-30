package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(restartCmd)
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the background weclaw process",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, live, err := inspectRuntimeState()
		if err != nil {
			return err
		}
		if len(live) > 0 {
			fmt.Printf("Stopping %d managed weclaw process(es)...\n", len(live))
			if err := stopManagedWeclaw(); err != nil {
				return err
			}
		} else {
			if err := stopManagedWeclaw(); err != nil {
				return err
			}
			fmt.Println("No managed weclaw process found; clearing stale runtime state")
		}

		apiAddr, err := resolveAPIAddr()
		if err != nil {
			return fmt.Errorf("failed to resolve API address: %w", err)
		}

		fmt.Println("Starting weclaw...")
		return runDaemon(false, apiAddr)
	},
}
