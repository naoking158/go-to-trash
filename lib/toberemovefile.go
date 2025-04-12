package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type ToBeMovedFile struct {
	From      string `json:"from"`
	To        string `json:"to"`
	MovedAt string `json:"moved_at"`
}

func NewToBeMovedFile(file File,
	trashDir string) *ToBeMovedFile {
	to := filepath.Join(trashDir, file.Name)

	// check file duplication
	_, err := os.Stat(to)
	if err == nil {
		// file duplicated
		fileExt := filepath.Ext(to)
		fileWOExt := strings.TrimSuffix(to, fileExt)

		// e.g., /path/to/foo.txt -> /path/to/foo.trash-20210101T000000Z0700.txt
		to = fmt.Sprintf("%v.trash-%v%v",
			fileWOExt,
			file.Timestamp.Format("20060102T150405Z0700"),
			fileExt,
		)
	}

	return &ToBeMovedFile{
		From:      file.Path,
		To:        to,
		MovedAt: file.Timestamp.Format("2006-01-02 15:04:05"),
	}
}

type RemovedFiles []ToBeMovedFile

func (files RemovedFiles) Sorted() RemovedFiles {
	slices.SortFunc(files, func(a ToBeMovedFile, b ToBeMovedFile) int {
		aTime, _ := time.Parse(time.DateTime, a.MovedAt)
		bTime, _ := time.Parse(time.DateTime, b.MovedAt)

		if aTime.Before(bTime) {
			return -1
		} else if aTime.After(bTime) {
			return 1
		} else {
			return 0
		}
	})
	return files
}
