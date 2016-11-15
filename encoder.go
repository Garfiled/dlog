package dlog

import (
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"
)

var textPool = sync.Pool{New: func() interface{} {
	return &textEncoder{
		bytes: make([]byte, 0, 1024),
	}
}}

type Encoder interface {
	KeyValue

	// Copy the encoder, ensuring that adding fields to the copy doesn't affect
	// the original.
	Clone() Encoder
	// Return the encoder to the appropriate sync.Pool. Unpooled encoder
	// implementations can no-op this method.
	Free()
	// Write the supplied message, level, and timestamp to the writer, along with
	// any accumulated context.
	WriteEntry(io.Writer, string, int, string, Level, time.Time) error
}

type textEncoder struct {
	bytes       []byte
	timeFmt     string
	firstNested bool
}

type LogMarshaler interface {
	MarshalLog(KeyValue) error
}

// NewTextEncoder creates a line-oriented text encoder whose output is optimized
// for human, rather than machine, consumption. By default, the encoder uses
// RFC3339-formatted timestamps.
func NewTextEncoder(options ...TextOption) *textEncoder {
	enc := textPool.Get().(*textEncoder)
	enc.truncate()
	enc.timeFmt = time.RFC3339
	for _, opt := range options {
		opt.apply(enc)
	}
	return enc
}

func (enc *textEncoder) Free() {
	textPool.Put(enc)
}

func (enc *textEncoder) AddString(key, val string) {
	enc.addKey(key)
	enc.bytes = append(enc.bytes, val...)
}

func (enc *textEncoder) AddBool(key string, val bool) {
	enc.addKey(key)
	enc.bytes = strconv.AppendBool(enc.bytes, val)
}

func (enc *textEncoder) AddInt(key string, val int) {
	enc.AddInt64(key, int64(val))
}

func (enc *textEncoder) AddInt64(key string, val int64) {
	enc.addKey(key)
	enc.bytes = strconv.AppendInt(enc.bytes, val, 10)
}

func (enc *textEncoder) AddUint(key string, val uint) {
	enc.AddUint64(key, uint64(val))
}

func (enc *textEncoder) AddUint64(key string, val uint64) {
	enc.addKey(key)
	enc.bytes = strconv.AppendUint(enc.bytes, val, 10)
}

func (enc *textEncoder) AddUintptr(key string, val uintptr) {
	enc.addKey(key)
	enc.bytes = append(enc.bytes, "0x"...)
	enc.bytes = strconv.AppendUint(enc.bytes, uint64(val), 16)
}

func (enc *textEncoder) AddFloat64(key string, val float64) {
	enc.addKey(key)
	enc.bytes = strconv.AppendFloat(enc.bytes, val, 'f', -1, 64)
}

func (enc *textEncoder) AddMarshaler(key string, obj LogMarshaler) error {
	enc.addKey(key)
	enc.firstNested = true
	enc.bytes = append(enc.bytes, '{')
	err := obj.MarshalLog(enc)
	enc.bytes = append(enc.bytes, '}')
	enc.firstNested = false
	return err
}

func (enc *textEncoder) AddObject(key string, obj interface{}) error {
	enc.AddString(key, fmt.Sprintf("%+v", obj))
	return nil
}

func (enc *textEncoder) Clone() Encoder {
	clone := textPool.Get().(*textEncoder)
	clone.truncate()
	clone.bytes = append(clone.bytes, enc.bytes...)
	clone.timeFmt = enc.timeFmt
	clone.firstNested = enc.firstNested
	return clone
}

func (enc *textEncoder) WriteEntry(sink io.Writer, caller string, line int, msg string, lvl Level, t time.Time) error {
	final := textPool.Get().(*textEncoder)
	final.truncate()
	enc.addLevel(final, lvl)
	enc.addTime(final, t)
	enc.addCaller(final, caller, line)
	enc.addMessage(final, msg)

	if len(enc.bytes) > 0 {
		final.bytes = append(final.bytes, ' ')
		final.bytes = append(final.bytes, enc.bytes...)
	}
	final.bytes = append(final.bytes, '\n')

	expectedBytes := len(final.bytes)
	n, err := sink.Write(final.bytes)
	final.Free()
	if err != nil {
		return err
	}
	if n != expectedBytes {
		return fmt.Errorf("incomplete write: only wrote %v of %v bytes", n, expectedBytes)
	}
	return nil
}

func (enc *textEncoder) truncate() {
	enc.bytes = enc.bytes[:0]
}

func (enc *textEncoder) addKey(key string) {
	lastIdx := len(enc.bytes) - 1
	if lastIdx >= 0 && !enc.firstNested {
		enc.bytes = append(enc.bytes, ' ')
	} else {
		enc.firstNested = false
	}
	enc.bytes = append(enc.bytes, key...)
	enc.bytes = append(enc.bytes, '=')
}

func (enc *textEncoder) addLevel(final *textEncoder, lvl Level) {
	final.bytes = append(final.bytes, '[')
	switch lvl {
	case DebugLevel:
		final.bytes = append(final.bytes, 'D')
	case InfoLevel:
		final.bytes = append(final.bytes, 'I')
	case WarnLevel:
		final.bytes = append(final.bytes, 'W')
	case ErrorLevel:
		final.bytes = append(final.bytes, 'E')
	case PanicLevel:
		final.bytes = append(final.bytes, 'P')
	case FatalLevel:
		final.bytes = append(final.bytes, 'F')
	default:
		final.bytes = strconv.AppendInt(final.bytes, int64(lvl), 10)
	}
	final.bytes = append(final.bytes, ']')
}
func (enc *textEncoder) addTime(final *textEncoder, t time.Time) {
	if enc.timeFmt == "" {
		return
	}
	final.bytes = append(final.bytes, ' ')
	final.bytes = t.AppendFormat(final.bytes, enc.timeFmt)
}

func (enc *textEncoder) addCaller(final *textEncoder, caller string, line int) {
	final.bytes = append(final.bytes, ` [`...)
	final.bytes = append(final.bytes, caller...)
	final.bytes = append(final.bytes, ':')
	final.bytes = strconv.AppendInt(final.bytes, int64(line), 10)
	final.bytes = append(final.bytes, ']')
}

func (enc *textEncoder) addMessage(final *textEncoder, msg string) {
	final.bytes = append(final.bytes, ' ')
	final.bytes = append(final.bytes, msg...)
}

// A TextOption is used to set options for a text encoder.
type TextOption interface {
	apply(*textEncoder)
}

type textOptionFunc func(*textEncoder)

func (opt textOptionFunc) apply(enc *textEncoder) {
	opt(enc)
}

// TextTimeFormat sets the format for log timestamps, using the same layout
// strings supported by time.Parse.
func TextTimeFormat(layout string) TextOption {
	return textOptionFunc(func(enc *textEncoder) {
		enc.timeFmt = layout
	})
}

// TextNoTime omits timestamps from the serialized log entries.
func TextNoTime() TextOption {
	return TextTimeFormat("")
}
