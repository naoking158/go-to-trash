package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"golang.org/x/sync/errgroup"
)

const DuplicatedTimeFormat = "20060102T150405Z0700"

type FileError error

var (
	ErrFileNotFound = errors.New("file not found")
	ErrFileInternal = errors.New("file internal error")
)

type ToBeMovedFile struct {
	From string
	To   string
}

type MovedFile struct {
	From    string
	To      string
	MovedAt time.Time
}

func NewToBeMovedFile(from, to string) ToBeMovedFile {
	return ToBeMovedFile{
		From: from,
		To:   to,
	}
}

func NewMovedFile(from, to string, movedAt time.Time) MovedFile {
	return MovedFile{
		From:    from,
		To:      to,
		MovedAt: movedAt,
	}
}

type ToBeMovedFiles []ToBeMovedFile

func (files ToBeMovedFiles) Move(isDryRun bool) ([]MovedFile, error) {
	movedFiles := make([]MovedFile, len(files))
	invalidPaths := make([]string, 0)
	uniqueFiles := resolveDuplicatesWithIndexSuffix(files)

	var eg errgroup.Group
	for i, f := range uniqueFiles {
		eg.Go(func() error {
			now := time.Now()
			from := f.From
			to, err := resolveDuplicateFilenameWithTimestamp(f.To, now)
			if err != nil {
				invalidPaths = append(invalidPaths, to)
				return errors.Wrap(err, "resolveDuplicateFilenameWithTimestamp")
			}

			if isDryRun {
				movedFiles[i] = NewMovedFile(from, to, now)
				return nil
			}

			// mkdirs
			if err := os.MkdirAll(filepath.Dir(to), 0777); err != nil {
				invalidPaths = append(invalidPaths, to)
				return errors.Wrap(err, "mkdirall")
			}

			// rename file
			if err := os.Rename(from, to); err != nil {
				invalidPaths = append(invalidPaths, to)
				return errors.Wrap(err, "os.rename")
			}

			movedFiles[i] = NewMovedFile(from, to, now)
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, errors.Wrapf(err, "failed to remove files: %v", invalidPaths)
	}

	return movedFiles, nil
}

func resolveDuplicatesWithIndexSuffix(files []ToBeMovedFile) []ToBeMovedFile {
	seen := make(map[string]int)
	unique := make([]ToBeMovedFile, len(files))

	for i, f := range files {
		count, ok := seen[f.To]
		seen[f.To]++

		// no duplicates
		if !ok {
			unique[i] = f
			continue
		}

		// duplicates found
		ext := filepath.Ext(f.To)
		name := strings.TrimSuffix(f.To, ext)
		newTo := fmt.Sprintf("%s(%d)%s", name, count, ext)

		unique[i] = NewToBeMovedFile(f.From, newTo)
	}

	return unique
}

func resolveDuplicateFilenameWithTimestamp(path string, now time.Time) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path, nil
	}

	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	newPath := fmt.Sprintf("%s.%s%s", base, now.Format(DuplicatedTimeFormat), ext)
	return newPath, nil
}
