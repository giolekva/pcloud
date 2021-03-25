package commands

import (
	"github.com/giolekva/pcloud/core/kg/app"
	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/giolekva/pcloud/core/kg/server"
	"github.com/giolekva/pcloud/core/kg/store/memory"
	"github.com/spf13/cobra"
)

// Command is an abstraction of the cobra Command
type Command = cobra.Command

// Run function starts the application
func Run(args []string) error {
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

// rootCmd is a command to run the server.
var rootCmd = &Command{
	Use:   "server",
	Short: "An example of the basic server",
	RunE:  serverCmdF,
}

func serverCmdF(command *cobra.Command, args []string) error {
	logger := log.NewLogger(&log.LoggerConfiguration{
		EnableConsole: true,
		ConsoleJSON:   true,
		ConsoleLevel:  "debug",
		EnableFile:    true,
		FileJSON:      true,
		FileLevel:     "debug",
		FileLocation:  "server.log",
	})
	config := model.NewConfig()

	st := memory.New()
	a := app.NewApp(st, logger)

	grpcServer := server.NewGRPCServer(logger, config, a)
	httpServer := server.NewHTTPServer(logger, config, a)

	servers := server.New(logger)
	servers.AddServers(grpcServer)
	servers.AddServers(httpServer)
	servers.Run()

	return nil
}
