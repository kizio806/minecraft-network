package networkcli

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type OperationLock struct{ file *os.File }

func AcquireLock(dir string) (*OperationLock, error) {
	path := filepath.Join(dir, ".network-operation.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open operation lock: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("another network operation is running")
	}
	return &OperationLock{file: f}, nil
}

func (l *OperationLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	return l.file.Close()
}
