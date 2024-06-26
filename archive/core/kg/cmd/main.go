package main

import (
	"os"

	"github.com/giolekva/pcloud/core/kg/cmd/commands"
)

func main() {
	if err := commands.Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
