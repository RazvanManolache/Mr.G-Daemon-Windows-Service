//make methods to read total disk space for each drive,  to calculate how much space a directory and all subfolders are taking up, and to symlink a directory to another location

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32            = windows.NewLazySystemDLL("kernel32.dll")
	getDiskFreeSpaceExW = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// GetTotalDiskSpace returns the total disk space in bytes

type DiskSpace struct {
	Total uint64
	Free  uint64
	Name  string
}

type Statfs_t struct {
	Type    int64
	Bsize   int64
	Blocks  int64
	Bfree   int64
	Bavail  int64
	Files   int64
	Ffree   int64
	Fsid    [8]int32
	Namelen int64
	Frsize  int64
	Flags   int64
	Spare   [4]int64
}

func GetTotalDiskSpaceForAllDrives() ([]DiskSpace, error) {
	drives, err := GetAllDrives()
	if err != nil {
		return nil, err
	}

	var totalSpace []DiskSpace
	for _, drive := range drives {
		total, free, err := GetTotalDiskSpace(drive)
		if err == nil {
			totalSpace = append(totalSpace, DiskSpace{Total: total, Free: free, Name: drive})
		}

	}
	return totalSpace, nil
}

func GetAllDrives() ([]string, error) {
	kernel32, err := syscall.LoadLibrary("Kernel32.dll")
	if err != nil {
		return nil, err
	}
	defer syscall.FreeLibrary(kernel32)

	getLogicalDrives, err := syscall.GetProcAddress(kernel32, "GetLogicalDrives")
	if err != nil {
		return nil, err
	}

	r1, _, e1 := syscall.SyscallN(uintptr(getLogicalDrives), 0, 0, 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			return nil, error(e1)
		} else {
			return nil, syscall.EINVAL
		}
	}

	var drives []string
	for i := 0; i < 26; i++ {
		if r1&(1<<uint(i)) != 0 {
			drive := fmt.Sprintf("%c:\\", 'A'+i)
			drives = append(drives, drive)
		}
	}
	return drives, nil
}

func GetTotalDiskSpace(drive string) (uint64, uint64, error) {
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64

	r1, _, err := getDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(drive))),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)

	if r1 == 0 {
		if err != nil {
			return 0, 0, err
		} else {
			return 0, 0, windows.APPMODEL_ERROR_NO_APPLICATION
		}
	}

	return totalNumberOfBytes, totalNumberOfFreeBytes, nil
}

// GetDirectorySize returns the size of a directory and all subfolders in bytes
func GetDirectorySize(directory string) (uint64, error) {
	var size uint64
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		size += uint64(info.Size())
		return nil
	})
	if err != nil {
		return size, err
	}
	return size, nil
}

// Symlink creates a symbolic link from source to destination
func Symlink(source, destination string) error {
	err := os.Symlink(source, destination)
	if err != nil {
		return err
	}
	return nil
}
