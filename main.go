package main

import (
	"log"
	"os"
)

func main() {
	config, err := NewConfig()
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
