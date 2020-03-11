// +build darwin,linux

package plugins

import (
	"fmt"
	"os"
	"syscall"
)

type FileRegistry struct {
	watchedFiles []*FileInfo
}

func (f *FileRegistry) IsNewPath(path string) bool {
	for _, file := range f.watchedFiles {
		if file.path == path {
			return false
		}
	}

	return true
}

func (f *FileRegistry) IsNewFileID(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return false, fmt.Errorf("failed to cast stat as syscall.Stat_t")
	}

	for _, file := range f.watchedFiles {
		if file.inode == stat.Ino && file.device == stat.Dev {
			return false, nil
		}
	}

	return true, nil
}

func (f *FileRegistry) IsNewFileContent(path string) bool {

}

type FileInfo struct {
	device int32
	inode  uint64
	path   string
}
