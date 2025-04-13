package lib

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/cockroachdb/errors"
)

const (
	HistoryFileName = "go-to-trash-history.json"
	RemovedAtFormat = time.RFC3339
)

var (
	ErrHistoryInvalid = errors.New("history is invalid")
)

type RemovedAt time.Time

func (r RemovedAt) Time() time.Time {
	return time.Time(r)
}

func (r RemovedAt) String() string {
	return time.Time(r).Format(RemovedAtFormat)
}

func (r RemovedAt) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(r).Format(RemovedAtFormat))
}

func (r *RemovedAt) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	t, err := time.Parse(RemovedAtFormat, str)
	if err != nil {
		return err
	}

	*r = RemovedAt(t)
	return nil
}

type HistoryEntry struct {
	From    string    `json:"from"`
	To      string    `json:"to"`
	Removed RemovedAt `json:"removed_at"`
}

func NewHistoryEntry(from, to string, removed RemovedAt) HistoryEntry {
	return HistoryEntry{
		From:    from,
		To:      to,
		Removed: removed,
	}
}

func NewHistoryEntriesFromMovedFiles(files []MovedFile) []HistoryEntry {
	entries := make([]HistoryEntry, len(files))
	for i, file := range files {
		entries[i] = HistoryEntry{
			From:    file.From,
			To:      file.To,
			Removed: RemovedAt(file.MovedAt),
		}
	}
	return entries
}

type HistoryEntries []HistoryEntry

func (entries HistoryEntries) Sorted() HistoryEntries {
	return slices.SortedStableFunc(slices.Values(entries), func(a, b HistoryEntry) int {
		if a.Removed.Time().Before(b.Removed.Time()) {
			return -1
		}
		if a.Removed.Time().After(b.Removed.Time()) {
			return 1
		}
		return 0
	})
}

type History struct {
	Path    string
	Entries []HistoryEntry
}

func NewHistory(path string, entries []HistoryEntry) *History {
	return &History{
		Path:    path,
		Entries: entries,
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

	var entries []HistoryEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry HistoryEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			// skip invalid lines
			continue
		}
		entries = append(entries, entry)
	}

	// handle potential scanning errors
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to scan history file")
	}

	return NewHistory(path, entries), nil
}

func (h *History) UpdateHistory(entries []HistoryEntry) error {
	// save
	if err := h.saveHistory(entries); err != nil {
		return errors.Wrap(err, "save history")
	}
	if err := h.SyncHistory(); err != nil {
		return errors.Wrap(err, "sync history")
	}

	return nil
}

func (h *History) saveHistory(entries []HistoryEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// check if the history file exists
	_, err := os.Stat(h.Path)
	if err != nil {
		if os.IsNotExist(err) {
			// if the file does not exist, create and write all history
			return writeEntriesToHistory(h.Path, entries)
		}
		return errors.Wrap(err, "failed to check history file existence")
	}

	// if the file exists, append only new entries
	return appendEntriesToHistory(h.Path, entries)
}

func (h *History) SyncHistory() error {
	uniqHist := UniqByKey(h.Entries, func(e HistoryEntry) string { return e.To })
	validFiles := make([]HistoryEntry, 0, len(uniqHist))

	// scan through the slice and remove unnecessary elements
	for _, entry := range uniqHist {
		_, err := os.Stat(entry.To)
		if err != nil {
			if os.IsNotExist(err) {
				// if the file does not exist, skip it (remove from history)
				continue
			}
			return errors.Wrap(err, "failed to check file existence")
		}

		// if the file exists, keep it by moving it to the front
		validFiles = append(validFiles, entry)
	}

	if len(h.Entries) == len(validFiles) {
		return nil
	}

	// update history files
	h.Entries = validFiles
	return writeEntriesToHistory(h.Path, h.Entries)
}

func writeEntriesToHistory(path string, entries []HistoryEntry) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to create history file")
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return errors.Wrap(err, "failed to marshal file entry")
		}
		if _, err := writer.Write(append(data, '\n')); err != nil {
			return errors.Wrap(err, "failed to write to history file")
		}
	}

	return writer.Flush()
}

func appendEntriesToHistory(path string, entries []HistoryEntry) error {
	uniqFiles := UniqByKey(entries, func(entry HistoryEntry) string { return entry.To })

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open history file for appending")
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	for _, entry := range uniqFiles {
		data, err := json.Marshal(entry)
		if err != nil {
			return errors.Wrap(err, "failed to marshal file entry")
		}
		if _, err := writer.Write(append(data, '\n')); err != nil {
			return errors.Wrap(err, "failed to write to history file")
		}
	}

	return writer.Flush()
}
