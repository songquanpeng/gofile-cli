package cmd

import (
	"github.com/spf13/cobra"
	"gofile-cli/common"
	"strconv"
)

func init() {
	rootCmd.AddCommand(pullCmd)
}

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull one or multi files by id",
	Long:  `Pull one or multi files by id`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := strconv.ParseUint(args[0], 10, 64)
		common.P2PRecvFileHandler(uint64(id))
	},
}
