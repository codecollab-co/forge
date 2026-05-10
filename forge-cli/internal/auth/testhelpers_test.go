package auth_test

import (
	"io/fs"
	"os"
)

type fileInfo struct{ mode fs.FileMode }

func statFile(path string) (fileInfo, error) {
	st, err := os.Stat(path)
	if err != nil {
		return fileInfo{}, err
	}
	return fileInfo{mode: st.Mode()}, nil
}
