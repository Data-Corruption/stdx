//go:build windows

package rlog

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// acquireRotationLock acquires an exclusive lock on a file in the specified directory.
// It returns a function to release the lock and an error if any occurs.
func acquireRotationLock(dir string) (func(), error) {
	file, err := os.OpenFile(filepath.Join(dir, ".rotate.lock"), os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	h := windows.Handle(file.Fd())
	overlapped := windows.Overlapped{}
	err = windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &overlapped)
	if err != nil {
		file.Close()
		return nil, err
	}
	return func() {
		windows.UnlockFileEx(h, 0, 1, 0, &overlapped)
		windows.CloseHandle(h)
		file.Close()
	}, nil
}
