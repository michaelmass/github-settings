package main

import (
	"log"

	"github.com/michaelmass/github-settings/pkg/github"
)

func main() {
	client := github.New("")

	settings, err := github.GetSettingsFromFile("settings.yml")

	if err != nil {
		log.Fatal(err)
	}

	err = client.Apply(settings)

	if err != nil {
		log.Fatal(err)
	}
}
