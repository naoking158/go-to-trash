package gototrash

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/naoking158/go-to-trash/domain"
	"github.com/naoking158/go-to-trash/util"
	"golang.org/x/sync/errgroup"
)

const Name = "gototrash"
const DefaultTrashDir = "~/.myTrash"

type Config struct {
	TrashDir string `json:"trashDir"`
}

func NewConfig() (*Config, error) {
	_, err := os.Stat("config.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// no error due to no config.json
			dir, _ := util.NormalizePath(DefaultTrashDir)
			return &Config{
				TrashDir: dir,
			}, nil
		}

		log.Println("failed to load config.json")
		return nil, errors.Wrap(err, "os.stat config.json")
	}

	f, err := os.Open("config.json")
	if err != nil {
		return nil, errors.Wrap(err, "open config.json")
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, errors.Wrap(err, "decode config.json")
	}

	normalizedTrashDir, err := util.NormalizePath(cfg.TrashDir)
	if err != nil {
		return nil, errors.Wrap(err, "normalize trashDir")
	}

	if _, err := os.Stat(normalizedTrashDir); err != nil {
		log.Printf("%v is not exist. Create it.", normalizedTrashDir)
		return nil, errors.Wrap(err, "os.stat trashDir")
	}
	cfg.TrashDir = normalizedTrashDir

	return &cfg, nil
}

// CLI is the main command line object
type CLI struct {
	Stdout, Stderr io.Writer
	TrashDir       string
}

func (cli *CLI) Run(args []string) int {
	var (
		dryrun bool
	)

	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.Stderr)

	flags.BoolVar(&dryrun, "dryrun", false, "no execute, just show what would be done")
	flags.BoolVar(&dryrun, "n", false, "alias for --dryrun")

	// Parse flags
	if err := flags.Parse(args[1:]); err != nil {
		// TODO: define exit code and use it
		return 1
	}

	parsedArgs := flags.Args()

	err := cli.remove(parsedArgs, dryrun)
	if err != nil {
		log.Println(err)
		return 1
	}

	// TODO: restore previous

	// TODO: save file info to inventory.json

	// TODO: restore file from inventory.json

	return 0
}

func (cli CLI) remove(paths []string, dryrun bool) error {
	files := make([]domain.ToBeRemoveFile, len(paths))
	invalidPaths := make([]string, 0)
	var eg errgroup.Group

	for i, path := range paths {
		eg.Go(func() error {
			f, err := domain.NewFile(path)
			if err != nil {
				invalidPaths = append(invalidPaths, path)
				return errors.Wrap(err, "new file")
			}

			toBeRemoveFile := domain.NewToBeRemoveFile(*f, cli.TrashDir)

			if dryrun {
				log.Printf("[DRYRUN] move `%v` to `%v`", toBeRemoveFile.From, toBeRemoveFile.To)
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

			log.Printf("move `%v` to `%v`", toBeRemoveFile.From, toBeRemoveFile.To)

			files[i] = *toBeRemoveFile
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return errors.Wrapf(err, "failed to remove files: %v", invalidPaths)
	}

	return nil
}
