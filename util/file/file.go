package file

import (
	"os"

	"github.com/cockroachdb/errors"
)

// Exist returns true if file at <path> exist.
func Exist(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}
