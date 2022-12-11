package main

import (
	"os"
	"strings"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	katanaapp "github.com/mokitanetwork/katana/app"
	appparams "github.com/mokitanetwork/katana/app/params"
	"github.com/mokitanetwork/katana/cmd/katanad/cmd"
)

func main() {
	rootCmd, _ := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, strings.ToUpper(appparams.Name), katanaapp.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
