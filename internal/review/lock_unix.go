//go:build !windows

package review

import (
	"errors"
	"os"
	"syscall"
)

func isBusy(err error) bool { return errors.Is(err, syscall.EWOULDBLOCK) }

func tryLock(f *os.File) error { return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) }
