package lib

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
)

const HistoryFileName = "go-to-trash-history.json"

var (
	ErrHistoryInvalid = errors.New("history is invalid")
)

type History struct {
	Path  string
	Files []ToBeMovedFile
}

func NewHistory(path string, files []ToBeMovedFile) *History {
	return &History{
		Path:  path,
		Files: files,
	}
}

func LoadHistory(trashDir string) (*History, error) {
	// history file path
	path := filepath.Join(trashDir, HistoryFileName)

	f, err := os.Open(path)
	if err != nil {
		// history file not found
		return NewHistory(path, nil), nil
	}
	defer f.Close()

	var files []ToBeMovedFile
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry ToBeMovedFile
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

func (h *History) UpdateHistory(files []ToBeMovedFile) error {
	// save
	if err := h.saveHistory(files); err != nil {
		return errors.Wrap(err, "save history")
	}

	return nil
}

func (h *History) saveHistory(files []ToBeMovedFile) error {
	if len(files) == 0 {
		return nil
	}

	// check if the history file exists
	_, err := os.Stat(h.Path)
	if err != nil {
		if os.IsNotExist(err) {
			// if the file does not exist, create and write all history
			return writeEntriesToHistory(h.Path, files)
		}
		return errors.Wrap(err, "failed to check history file existence")
	}

	// if the file exists, append only new entries
	return appendEntriesToHistory(h.Path, files)
}

func (h *History) SyncHistory() error {
	uniqHist := UniqByKey(h.Files, func(f ToBeMovedFile) string { return f.To })
	validFiles := make([]ToBeMovedFile, 0, len(uniqHist))

	// scan through the slice and remove unnecessary elements
	for _, file := range uniqHist {
		_, err := os.Stat(file.To)
		if err != nil {
			if os.IsNotExist(err) {
				// if the file does not exist, skip it (remove from history)
				continue
			}
			return errors.Wrap(err, "failed to check file existence")
		}

		// if the file exists, keep it by moving it to the front
		validFiles = append(validFiles, file)
	}

	if len(h.Files) == len(validFiles) {
		return nil
	}

	// update history files
	h.Files = validFiles
	return writeEntriesToHistory(h.Path, h.Files)
}

func writeEntriesToHistory(path string, files []ToBeMovedFile) error {
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

func appendEntriesToHistory(path string, files []ToBeMovedFile) error {
	uniqFiles := UniqByKey(files, func(file ToBeMovedFile) string { return file.To })

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open history file for appending")
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	for _, file := range uniqFiles {
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
