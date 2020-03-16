// +build windows

package fileinput

import (
	"fmt"
	"os"
	"syscall"
)

func safeDevInode(file *os.File) (int32, uint64, error) {
	if file == nil {
		return 0, 0, fmt.Errorf("file unopened")
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, 0, err
	}

	switch sys := fileInfo.Sys().(type) {
	case *syscall.Win32FileAttributeData:
		return sys.Dev, sys.Ino, nil
	default:
		return 0, 0, fmt.Errorf("cannot use fileinfo of type %T", fileInfo.Sys())
	}
}
