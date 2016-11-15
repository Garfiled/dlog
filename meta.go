package dlog

import (
	"os"
	"sync/atomic"
)

// Meta is implementation-agnostic state management for Loggers. Most Logger
// implementations can reduce the required boilerplate by embedding a Meta.
//
// Note that while the level-related fields and methods are safe for concurrent
// use, the remaining fields are not.
type Meta struct {
	Development bool
	Encoder     Encoder
	Output      WriteSyncer
	ErrorOutput WriteSyncer

	lvl int32
}

// MakeMeta returns a new meta struct with sensible defaults: logging at
// InfoLevel, development mode off, and writing to standard error and standard
// out.
func MakeMeta(enc Encoder) Meta {
	return Meta{
		lvl:         int32(InfoLevel),
		Encoder:     enc,
		Output:      newLockedWriteSyncer(os.Stdout),
		ErrorOutput: newLockedWriteSyncer(os.Stderr),
	}
}

// Level returns the minimum enabled log level. It's safe to call concurrently.
func (m Meta) Level() Level {
	return Level(atomic.LoadInt32(&m.lvl))
}

// SetLevel atomically alters the the logging level for this Meta and all its
// clones.
func (m Meta) SetLevel(lvl Level) {
	atomic.StoreInt32(&m.lvl, int32(lvl))
}

// Clone creates a copy of the meta struct. It deep-copies the encoder, but not
// the hooks (since they rarely change).
func (m Meta) Clone() Meta {
	m.Encoder = m.Encoder.Clone()
	return m
}
