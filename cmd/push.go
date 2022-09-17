package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pushCmd)
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push one or multi local files for sharing",
	Long:  `Push one or multi local files for others to pull`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Ready to push files:")
		for i, s := range args {
			fmt.Println("\t", i, s)
		}
	},
}
