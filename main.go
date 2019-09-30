package main

import (
	"github.com/michaelmass/github-settings/cmd"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func main() {
	log.SetFormatter(&prefixed.TextFormatter{})
	cmd.Execute()
}
