// Platform helpers for FIFO creation and non-blocking open on macOS / Darwin.
//go:build darwin

package fifo

import (
	"context"
	"fmt"
	"syscall"
)

// mkfifo calls the libc mkfifo(2) via Go's syscall package.
func mkfifo(path string, perm uint32) error {
	return syscall.Mkfifo(path, perm)
}

// openFIFORaw opens path with O_RDWR | O_NONBLOCK and returns the raw fd.
//
// O_RDWR is used instead of O_RDONLY to avoid ENXIO on macOS when no writer
// is connected.  O_NONBLOCK ensures the open(2) call itself never blocks.
// The caller is expected to use select(2)/poll(2) to wait for data.
func openFIFORaw(ctx context.Context, path string) (int, error) {
	type result struct {
		fd  int
		err error
	}
	ch := make(chan result, 1)

	go func() {
		fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_NONBLOCK, 0)
		ch <- result{fd, err}
	}()

	select {
	case <-ctx.Done():
		return -1, fmt.Errorf("context cancelled")
	case r := <-ch:
		return r.fd, r.err
	}
}
