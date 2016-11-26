package main

import (
	"os"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app     = kingpin.New("cloudo", "Cloud fast provisioning scripts")
	verbose = app.Flag("v", "verbose mode").Bool()

	aws    = app.Command("aws", "AWS API")
	create = aws.Command("create", "provisions AWS EC2 instances with new VPC/Subnet/IG/Security Group with enabled ssh connection")
)

func main() {
	switch v, _ := app.Parse(os.Args[1:]); v {
	case "aws create":

	}
}
