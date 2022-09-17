package cmd

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gofile-cli/common"
	"os"
)

var rootCmd = &cobra.Command{
	Use:     "gofile-cli",
	Version: common.Version,
	Short:   "CLI tool for Go File",
	Long: `CLI tool to interact with the Go File server.
Complete documentation is available at https://github.com/songquanpeng/gofile-cli`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

var cfgFile string
var cfgFileName = ".gofile-cli"

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/"+cfgFileName+")")
	rootCmd.PersistentFlags().StringP("server", "s", "", "Go File server address")
	rootCmd.PersistentFlags().Bool("viper", true, "Use Viper for configuration")
	_ = viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server"))
	_ = viper.BindPFlag("useViper", rootCmd.PersistentFlags().Lookup("viper"))
}

func initConfig() {
	// Don't forget to read config either from cfgFile or from home directory!
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".cobra" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(cfgFileName)
		_ = viper.SafeWriteConfig()
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
