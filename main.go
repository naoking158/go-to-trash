package main

import (
	"log"
	"os"

	"github.com/naoking158/go-to-trash/lib"
)

func main() {
	config, err := lib.NewConfig()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	cli := CLI{
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		TrashDir: config.TrashDir,
	}
	os.Exit(cli.Run(os.Args))
}
