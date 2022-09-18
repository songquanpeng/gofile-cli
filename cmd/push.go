package cmd

import (
	"github.com/spf13/cobra"
	"gofile-cli/common"
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
		common.P2PSendFileHandler(args)
	},
}
