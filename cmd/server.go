package cmd

import "github.com/spf13/cobra"

var showCmd = &cobra.Command{
	Use:   "server",
	Short: "Starts web server",
	Long:  `Starts a local web server that let's agents script `,
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func init() {
	createCmd.Flags().String("port", ":3333", "The default port to run on")
	rootCmd.AddCommand(showCmd)
}
