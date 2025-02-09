package main

import (
	"log"
	"os"

	trash "github.com/naoking158/go-to-trash"
)

func main() {
	config, err := trash.NewConfig()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	cli := trash.CLI{
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		TrashDir: config.TrashDir,
	}
	os.Exit(cli.Run(os.Args))
}
