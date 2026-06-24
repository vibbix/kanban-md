package cmd

import (
	"github.com/antopolskiy/kanban-md/internal/api"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Starts the web server",
	Long:  `Starts a local REST API server powered by OpenAPI 3.1.`,
	Args:  cobra.NoArgs,
	RunE:  runServerCmd,
}

func init() {
	serverCmd.Flags().String("port", ":3333", "The port to run the server on (e.g. :3333)")
	rootCmd.AddCommand(serverCmd)
}

func runServerCmd(cmd *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	port, _ := cmd.Flags().GetString("port")
	
	// Delegate all server logic to the internal/api package
	return api.Start(cfg, port)
}
