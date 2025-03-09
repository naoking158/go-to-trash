package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type RemovedFile struct {
	From      string `json:"from"`
	To        string `json:"to"`
	RemovedAt string `json:"removed_at"`
}

func NewToBeRemoveFile(file File, trashDir string) *RemovedFile {
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

	return &RemovedFile{
		From:      file.Path,
		To:        to,
		RemovedAt: file.Timestamp.Format("2006-01-02 15:04:05"),
	}
}

type RemovedFiles []RemovedFile

func (files RemovedFiles) Sorted() RemovedFiles {
	slices.SortFunc(files, func(a RemovedFile, b RemovedFile) int {
		aTime, _ := time.Parse(time.DateTime, a.RemovedAt)
		bTime, _ := time.Parse(time.DateTime, b.RemovedAt)

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
