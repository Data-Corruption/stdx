//go:build linux || darwin || freebsd || netbsd || openbsd

package rlog

import (
	"os"
	"path/filepath"
	"syscall"
)

// acquireRotationLock acquires an exclusive lock on a file in the specified directory.
// It returns a function to release the lock and an error if any occurs.
func acquireRotationLock(dir string) (func(), error) {
	f, err := os.OpenFile(filepath.Join(dir, ".rotate.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}
