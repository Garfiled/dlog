package dlog

import (
	"io"
	"sync"
)

// A WriteFlusher is an io.Writer that can also flush any buffered data.
type WriteFlusher interface {
	io.Writer
	Flush() error
}

// A WriteSyncer is an io.Writer that can also flush any buffered data. Note
// that *os.File (and thus, os.Stderr and os.Stdout) implement WriteSyncer.
type WriteSyncer interface {
	io.Writer
	Sync() error
}

// AddSync converts an io.Writer to a WriteSyncer. It attempts to be
// intelligent: if the concrete type of the io.Writer implements WriteSyncer or
// WriteFlusher, we'll use the existing Sync or Flush methods. If it doesn't,
// we'll add a no-op Sync method.
func AddSync(w io.Writer) WriteSyncer {
	switch w := w.(type) {
	case WriteSyncer:
		return w
	case WriteFlusher:
		return flusherWrapper{w}
	default:
		return writerWrapper{w}
	}
}

type lockedWriteSyncer struct {
	sync.Mutex
	ws WriteSyncer
}

func newLockedWriteSyncer(ws WriteSyncer) WriteSyncer {
	return &lockedWriteSyncer{ws: ws}
}

func (s *lockedWriteSyncer) Write(bs []byte) (int, error) {
	s.Lock()
	n, err := s.ws.Write(bs)
	s.Unlock()
	return n, err
}

func (s *lockedWriteSyncer) Sync() error {
	s.Lock()
	err := s.ws.Sync()
	s.Unlock()
	return err
}

type writerWrapper struct {
	io.Writer
}

func (w writerWrapper) Sync() error {
	return nil
}

type flusherWrapper struct {
	WriteFlusher
}

func (f flusherWrapper) Sync() error {
	return f.Flush()
}
