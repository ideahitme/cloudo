package main

import (
	"os"

	"github.com/ideahitme/cloudo/awsclient"

	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	var debug bool
	app := kingpin.New("cloudo", "Cloud fast provisioning script")
	app.Flag("debug", "debug mode").BoolVar(&debug)

	awsclient.ReadFlags(app)

	kingpin.MustParse(app.Parse(os.Args[1:]))
}
