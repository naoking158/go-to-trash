package lib

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
)

const historyFileName = "go-to-trash-history.json"

var (
	ErrHistoryInvalid = errors.New("history is invalid")
)

type History struct {
	Path  string
	Files []ToBeRemoveFile
}

func NewHistory(path string, files []ToBeRemoveFile) *History {
	return &History{
		Path:  path,
		Files: files,
	}
}

func LoadHistory(trashDir string) (*History, error) {
	// history file path
	path := filepath.Join(trashDir, historyFileName)

	f, err := os.Open(path)
	if err != nil {
		// history file not found
		return NewHistory(path, nil), nil
	}
	defer f.Close()

	var files []ToBeRemoveFile
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry ToBeRemoveFile
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			// skip invalid lines
			continue
		}
		files = append(files, entry)
	}

	// handle potential scanning errors
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to scan history file")
	}

	return NewHistory(path, files), nil
}

func (i *History) UpdateHistory(files []ToBeRemoveFile) error {
	// sync
	if err := i.syncHistory(); err != nil {
		return errors.Wrap(err, "sync history")
	}

	// save
	if err := i.saveHistory(files); err != nil {
		return errors.Wrap(err, "save history")
	}

	return nil
}

func (i *History) saveHistory(files []ToBeRemoveFile) error {
	// sync history to remove non-existing records
	if err := i.syncHistory(); err != nil {
		return errors.Wrap(err, "failed to sync history")
	}

	if len(files) == 0 {
		return nil
	}

	// check if the history file exists
	_, err := os.Stat(i.Path)
	if err != nil {
		if os.IsNotExist(err) {
			// if the file does not exist, create and write all history
			return writeEntriesToHistory(i.Path, append(i.Files, files...))
		}
		return errors.Wrap(err, "failed to check history file existence")
	}

	// if the file exists, append only new entries
	return appendEntriesToHistory(i.Path, files)
}

func (i *History) syncHistory() error {
	// index to track the number of valid entries
	validCount := 0

	// scan through the slice and remove unnecessary elements
	for _, file := range i.Files {
		if _, err := os.Stat(file.To); err != nil {
			if os.IsNotExist(err) {
				// if the file does not exist, skip it (remove from history)
				continue
			}
			return errors.Wrap(err, "failed to check file existence")
		}

		// if the file exists, keep it by moving it to the front
		i.Files[validCount] = file
		validCount++
	}

	// keep only the valid entries
	i.Files = i.Files[:validCount]

	return nil
}

func writeEntriesToHistory(path string, files []ToBeRemoveFile) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to create history file")
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	for _, file := range files {
		data, err := json.Marshal(file)
		if err != nil {
			return errors.Wrap(err, "failed to marshal file entry")
		}
		if _, err := writer.Write(append(data, '\n')); err != nil {
			return errors.Wrap(err, "failed to write to history file")
		}
	}

	return writer.Flush()
}

func appendEntriesToHistory(path string, files []ToBeRemoveFile) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open history file for appending")
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	for _, file := range files {
		data, err := json.Marshal(file)
		if err != nil {
			return errors.Wrap(err, "failed to marshal file entry")
		}
		if _, err := writer.Write(append(data, '\n')); err != nil {
			return errors.Wrap(err, "failed to write to history file")
		}
	}

	return writer.Flush()
}
