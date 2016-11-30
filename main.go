package main

import (
	"fmt"
	"os"

	"github.com/ideahitme/cloudo/awsclient"

	"gopkg.in/alecthomas/kingpin.v2"
)

var Version string

func main() {
	var debug bool
	app := kingpin.New("cloudo", "Cloud fast provisioning script")
	app.Flag("debug", "debug mode").BoolVar(&debug)
	app.Command("version", "cloudo version").Action(func(ctx *kingpin.ParseContext) error {
		fmt.Printf("Current version: %s\n", Version)
		return nil
	})

	awsclient.ReadFlags(app)

	kingpin.MustParse(app.Parse(os.Args[1:]))
}
