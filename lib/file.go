package lib

import (
	"os"
	"time"

	"github.com/cockroachdb/errors"
)

type FileError error

var (
	ErrFileNotFound = errors.New("file not found")
	ErrFileInternal = errors.New("file internal error")
)

type File struct {
	Name      string
	Path      string
	Timestamp time.Time
}

func NewFile(path string) (*File, error) {
	normPath, err := NormalizePath(path)
	if err != nil {
		return nil, errors.Wrapf(errors.Join(err, ErrFileInternal), "normalize path: %v", path)
	}

	// check path existance
	f, err := os.Stat(path)
	if err != nil {
		return nil, errors.Wrap(errors.Join(err, ErrFileNotFound), "os stat")
	}

	return &File{
		Name:      f.Name(),
		Path:      normPath,
		Timestamp: time.Now().Local(),
	}, nil
}
