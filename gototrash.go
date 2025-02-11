package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"golang.org/x/sync/errgroup"
)

const Name = "gototrash"

// CLI is the main command line object
type CLI struct {
	Stdout, Stderr io.Writer
	TrashDir       string
}

func (cli *CLI) Run(args []string) int {
	var (
		dryrun bool
		verbose bool
	)

	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.Stderr)

	flags.BoolVar(&dryrun, "dryrun", false, "no execute, just show what would be done")
	flags.BoolVar(&dryrun, "n", false, "alias for --dryrun")

	flags.BoolVar(&verbose, "verbose", false, "show verbose output")
	flags.BoolVar(&verbose, "v", false, "alias for --verbose")

	// Parse flags
	if err := flags.Parse(args[1:]); err != nil {
		// TODO: define exit code and use it
		fmt.Fprintf(cli.Stderr, "failed to parse flags: %v\n", err)
		return 1
	}

	if verbose {
		log.SetOutput(cli.Stderr)
	} else {
		log.SetOutput(io.Discard)
	}

	paths := flags.Args()
	err := cli.remove(paths, dryrun)
	if err != nil {
		log.Println(err)
		fmt.Fprintf(cli.Stderr, "failed to remove: %v\n", err)
		return 1
	}

	// TODO: restore previous
	
	// TODO: save file info to inventory.json

	// TODO: restore file from inventory.json

	return 0
}

func (cli CLI) remove(paths []string, dryrun bool) error {
	files := make([]ToBeRemoveFile, len(paths))
	invalidPaths := make([]string, 0)

	var eg errgroup.Group

	for i, path := range paths {
		eg.Go(func() error {
			f, err := NewFile(path)
			if err != nil {
				if errors.Is(err, ErrFileNotFound) {
					fmt.Fprintf(cli.Stderr, "file not found: %v\n", path)
				}
				
				invalidPaths = append(invalidPaths, path)
				return errors.Wrap(err, "new file")
			}

			toBeRemoveFile := NewToBeRemoveFile(*f, cli.TrashDir)

			if dryrun {
				fmt.Fprintf(cli.Stdout, "[DRYRUN] move `%v` to `%v`\n", toBeRemoveFile.From, toBeRemoveFile.To)
				return nil
			}

			// makedirs
			if err := os.MkdirAll(filepath.Dir(toBeRemoveFile.To), 0777); err != nil {
				invalidPaths = append(invalidPaths, path)
				return errors.Wrap(err, "mkdirall")
			}

			// rename file
			if err := os.Rename(toBeRemoveFile.From, toBeRemoveFile.To); err != nil {
				invalidPaths = append(invalidPaths, path)
				return errors.Wrap(err, "os.rename")
			}

			fmt.Fprintf(cli.Stdout, "move `%v` to `%v`\n", toBeRemoveFile.From, toBeRemoveFile.To)

			files[i] = *toBeRemoveFile
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return errors.Wrapf(err, "failed to remove files: %v", invalidPaths)
	}

	return nil
}
