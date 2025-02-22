package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/naoking158/go-to-trash/lib"
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
		dryrun  bool
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

	history, err := lib.LoadHistory(cli.TrashDir)
	if err != nil {
		log.Println(err)
		fmt.Fprintf(cli.Stderr, "failed to load history: %v\n", err)
		return 1
	}
	if err := history.SyncHistory(); err != nil {
		log.Println(err)
		fmt.Fprintf(cli.Stderr, "failed to sync history: %v\n", err)
		return 1
	}

	removedFiles, err := cli.remove(paths, dryrun)
	if err != nil {
		log.Println(err)
		fmt.Fprintf(cli.Stderr, "failed to remove: %v\n", err)
		return 1
	}

	if err := history.UpdateHistory(removedFiles); err != nil {
		log.Println(err)
		fmt.Fprintf(cli.Stderr, "failed to update history: %v\n", err)
		return 1
	}

	return 0
}

func (cli CLI) remove(paths []string, dryrun bool) ([]lib.RemovedFile, error) {
	files := make([]lib.RemovedFile, len(paths))
	invalidPaths := make([]string, 0)

	var eg errgroup.Group

	for i, path := range paths {
		eg.Go(func() error {
			f, err := lib.NewFile(path)
			if err != nil {
				if errors.Is(err, lib.ErrFileNotFound) {
					fmt.Fprintf(cli.Stderr, "file not found: %v\n", path)
				}

				invalidPaths = append(invalidPaths, path)
				return errors.Wrap(err, "new file")
			}

			toBeRemoveFile := lib.NewToBeRemoveFile(*f, cli.TrashDir)

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
		return nil, errors.Wrapf(err, "failed to remove files: %v", invalidPaths)
	}

	return files, nil
}
