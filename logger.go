package dlog

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"
)

// For tests.
var _exit = os.Exit

// A Logger enables leveled, structured logging. All methods are safe for
// concurrent use.
type Logger interface {
	// Check the minimum enabled log level.
	Level() Level
	// Change the level of this logger, as well as all its ancestors and
	// descendants. This makes it easy to change the log level at runtime
	// without restarting your application.
	SetLevel(Level)

	// Create a child logger, and optionally add some context to that logger.
	With(...Field) Logger

	// Check returns a CheckedMessage if logging a message at the specified level
	// is enabled. It's a completely optional optimization; in high-performance
	// applications, Check can help avoid allocating a slice to hold fields.
	//
	// See CheckedMessage for an example.
	// Check(Level, string) *CheckedMessage

	// Log a message at the given level. Messages include any context that's
	// accumulated on the logger, as well as any fields added at the log site.
	Log(Level, string, ...Field)
	Debug(string, ...Field)
	Info(string, ...Field)
	Warn(string, ...Field)
	Error(string, ...Field)
	Panic(string, ...Field)
	Fatal(string, ...Field)
	// If the logger is in development mode (via the Development option), DFatal
	// logs at the Fatal level. Otherwise, it logs at the Error level.
	DFatal(string, ...Field)
}

type logger struct{ Meta }

var (
	dlogger    *logger
	syncFile   *os.File
	syncTicker *time.Ticker
)

// New constructs a logger that uses the provided encoder. By default, the
// logger will write Info logs or higher to standard out. Any errors during logging
// will be written to standard error.
//
// Options can change the log level, the output location, the initial fields
// that should be added as context, and many other behaviors.
func Init(filepath string) error {
	f, err := os.OpenFile(filepath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		return err
	}
	dlogger = &logger{
		Meta: MakeMeta(NewTextEncoder()),
	}
	dlogger.Output = newLockedWriteSyncer(f)
	syncFile = f
	go syncEntry(f)
	return nil
}

func syncEntry(f *os.File) {
	syncTicker = time.NewTicker(time.Millisecond * 500)
	for range syncTicker.C {
		f.Sync()
	}
}

func Debug(msg string, fields ...Field) {
	dlogger.log(DebugLevel, msg, fields)
}

func Info(msg string, fields ...Field) {
	dlogger.log(InfoLevel, msg, fields)
}

func Warn(msg string, fields ...Field) {
	dlogger.log(WarnLevel, msg, fields)
}

func Error(msg string, fields ...Field) {
	dlogger.log(ErrorLevel, msg, fields)
}

func Panic(msg string, fields ...Field) {
	dlogger.log(PanicLevel, msg, fields)
	panic(msg)
}

func Fatal(msg string, fields ...Field) {
	dlogger.log(FatalLevel, msg, fields)
	_exit(1)
}

func Close() {
	syncTicker.Stop()
	syncFile.Sync()
	syncFile.Close()
}

func (log *logger) log(lvl Level, msg string, fields []Field) {
	if !(lvl >= log.Level()) {
		return
	}

	temp := log.Encoder.Clone()
	addFields(temp, fields)

	caller, line := CallerName1()

	if err := temp.WriteEntry(log.Output, caller, line, msg, lvl, time.Now()); err != nil {
		log.internalError(err.Error())
	}
	temp.Free()
}

func (log *logger) internalError(msg string) {
	fmt.Fprintln(log.ErrorOutput, msg)
	log.ErrorOutput.Sync()
}

// func callerName() string {
// 	var pc uintptr
// 	var file string
// 	var line int
// 	var ok bool
// 	if pc, file, line, ok = runtime.Caller(3); !ok {
// 		return ""
// 	}
// 	name := runtime.FuncForPC(pc).Name()
// 	res := "[" + path.Base(file) + ":" + strconv.Itoa(line) + "]" + name
// 	return res
// }

func CallerName() string {
	var file string
	var line int
	var ok bool
	if _, file, line, ok = runtime.Caller(3); !ok {
		return ""
	}
	res := "[" + path.Base(file) + ":" + strconv.Itoa(line) + "]"
	return res
}

func CallerName1() (string, int) {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return "", 0
	}
	return path.Base(file), line
}
