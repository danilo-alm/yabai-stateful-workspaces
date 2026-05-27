// Package fifo implements a resilient reader for a Unix named pipe (FIFO).
//
// Design notes
// ────────────
// A FIFO on macOS does not support SetDeadline and closing an os.File from
// another goroutine does not reliably unblock a concurrent Read.  To work
// around both limitations this package:
//
//  1. Opens the FIFO via syscall.Open (raw fd) so Go's runtime poller never
//     touches it and the blocking-mode switch caused by os.File.Fd() is
//     avoided.
//
//  2. Uses the self-pipe trick + syscall.Select to wait for data. A cancel
//     pipe is created at the start of each read session; when ctx is
//     cancelled the write end is closed, which makes the read end readable
//     and unblocks Select immediately.
//
//  3. Reopens the FIFO after each EOF so the daemon survives multiple
//     separate skhd invocations.
package fifo

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"syscall"
)

// Reader continuously reads single-byte tokens from a FIFO file and sends
// them on the Lines channel.
type Reader struct {
	Path  string
	Lines chan string
}

// New creates a Reader for the FIFO at path. The buffered channel absorbs
// short bursts of rapid keypresses without dropping them.
func New(path string) *Reader {
	return &Reader{
		Path:  path,
		Lines: make(chan string, 16),
	}
}

// ensureFIFO creates the FIFO at r.Path if it does not already exist.
// If a non-FIFO file exists at the path (e.g. a regular file created by a
// shell redirect before the daemon started), it is removed and replaced.
func (r *Reader) ensureFIFO() error {
	info, err := os.Stat(r.Path)
	if err == nil {
		if info.Mode()&os.ModeNamedPipe != 0 {
			return nil // already a FIFO, nothing to do
		}
		// A regular file (or something else) is in the way — remove it.
		slog.Warn("path exists but is not a FIFO, removing it", "path", r.Path)
		if err := os.Remove(r.Path); err != nil {
			return fmt.Errorf("remove non-FIFO at %s: %w", r.Path, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat %s: %w", r.Path, err)
	}
	if err := mkfifo(r.Path, 0o600); err != nil {
		return fmt.Errorf("mkfifo %s: %w", r.Path, err)
	}
	slog.Info("FIFO created", "path", r.Path)
	return nil
}

// Run starts the read loop. It blocks until ctx is cancelled.
// On return, the Lines channel is closed and the FIFO file is removed.
func (r *Reader) Run(ctx context.Context) error {
	if err := r.ensureFIFO(); err != nil {
		return err
	}

	defer func() {
		close(r.Lines)
		if err := os.Remove(r.Path); err != nil && !os.IsNotExist(err) {
			slog.Warn("could not remove FIFO", "path", r.Path, "err", err)
		} else {
			slog.Info("FIFO removed", "path", r.Path)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		fd, err := openFIFORaw(ctx, r.Path)
		if err != nil {
			// ctx was cancelled while waiting for a writer.
			return nil
		}

		slog.Debug("FIFO writer connected")
		if err := r.readBytes(ctx, fd); err != nil {
			slog.Error("FIFO read error", "err", err)
		}
		syscall.Close(fd)
		slog.Debug("FIFO writer disconnected, reopening")
	}
}

// readBytes drains fd byte-by-byte, dispatching each printable byte as a token.
//
// Cancellation is handled via a cancel pipe: a goroutine closes the write end
// when ctx fires, making the read end readable and immediately waking Select.
// Both fds are raw syscall ints so Go's runtime poller is never involved.
func (r *Reader) readBytes(ctx context.Context, fifoFd int) error {
	var pipeFds [2]int
	if err := syscall.Pipe(pipeFds[:]); err != nil {
		return fmt.Errorf("create cancel pipe: %w", err)
	}
	cancelR, cancelW := pipeFds[0], pipeFds[1]
	defer syscall.Close(cancelR)

	// Close the write end when ctx is cancelled to unblock Select below.
	go func() {
		<-ctx.Done()
		syscall.Close(cancelW)
	}()

	nfds := fifoFd + 1
	if cancelR+1 > nfds {
		nfds = cancelR + 1
	}

	buf := make([]byte, 4096)

	for {
		var readSet syscall.FdSet
		fdSet(&readSet, fifoFd)
		fdSet(&readSet, cancelR)

		// 100 ms timeout as a safety net; the cancel pipe handles the fast path.
		tv := syscall.Timeval{Sec: 0, Usec: 100_000}

		if err := syscall.Select(nfds, &readSet, nil, nil, &tv); err != nil {
			if err == syscall.EINTR {
				continue
			}
			return fmt.Errorf("select: %w", err)
		}

		// Cancel pipe became readable — ctx was cancelled.
		if fdIsSet(&readSet, cancelR) {
			return nil
		}

		// Safety: also check ctx on the timeout path.
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if !fdIsSet(&readSet, fifoFd) {
			continue
		}

		slog.Debug("FIFO ready to read")
		n, err := syscall.Read(fifoFd, buf)
		slog.Debug("read from FIFO", "n", n, "err", err)
		for _, b := range buf[:n] {
			slog.Debug("byte received", "byte", b, "char", string([]byte{b}))
			if b <= ' ' { // skip whitespace / newlines / control chars
				continue
			}
			select {
			case r.Lines <- string([]byte{b}):
			case <-ctx.Done():
				return nil
			}
		}
		if err != nil {
			if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
				continue
			}
			return fmt.Errorf("read fifo: %w", err)
		}
		if n == 0 {
			// EOF: all writers closed. Return so Run() reopens.
			return nil
		}
	}
}

// fdSet sets bit fd in a syscall.FdSet.
func fdSet(set *syscall.FdSet, fd int) {
	set.Bits[fd/64] |= 1 << (uint(fd) % 64)
}

// fdIsSet reports whether bit fd is set in a syscall.FdSet.
func fdIsSet(set *syscall.FdSet, fd int) bool {
	return set.Bits[fd/64]&(1<<(uint(fd)%64)) != 0
}
