package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"path/filepath"

	"github.com/naoking158/go-to-trash/lib"
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
		restore bool
	)

	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.Stderr)

	flags.BoolVar(&dryrun, "dryrun", false, "no execute, just show what would be done")
	flags.BoolVar(&dryrun, "n", false, "alias for --dryrun")

	flags.BoolVar(&verbose, "verbose", false, "show verbose output")
	flags.BoolVar(&verbose, "v", false, "alias for --verbose")

	flags.BoolVar(&restore, "restore", false, "restore files from trash")

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

	if restore {
		if err := lib.Restore(history.Entries); err != nil {
			log.Println(err)
			fmt.Fprintf(cli.Stderr, "there's been an error: %v", err)
			return 1
		}

		return 0
	}

	removedFiles, err := cli.remove(paths, dryrun)
	if err != nil {
		log.Println(err)
		fmt.Fprintf(cli.Stderr, "failed to remove: %v\n", err)
		return 1
	}

	for _, f := range removedFiles {
		fmt.Fprintf(cli.Stdout, "removed: %s → %s\n", f.From, f.To)
	}

	if err := history.UpdateHistory(lib.NewHistoryEntriesFromMovedFiles(removedFiles)); err != nil {
		log.Println(err)
		fmt.Fprintf(cli.Stderr, "failed to update history: %v\n", err)
		return 1
	}

	return 0
}

func (cli *CLI) remove(paths []string, dryrun bool) ([]lib.MovedFile, error) {
	toBeMovedFiles := make(lib.ToBeMovedFiles, len(paths))
	for i, path := range paths {
		from, err := lib.ValidatePath(path)
		if err != nil {
			log.Println(err)
			fmt.Fprintf(cli.Stderr, "failed to validate path: %v\n", err)
			return nil, fmt.Errorf("failed to validate path: %w", err)
		}

		to := filepath.Join(cli.TrashDir, filepath.Base(path))

		toBeMovedFiles[i] = lib.NewToBeMovedFile(from, to)
	}

	movedFiles, err := toBeMovedFiles.Move(dryrun)
	if err != nil {
		return nil, fmt.Errorf("failed to move files: %w", err)
	}

	return movedFiles, nil
}
