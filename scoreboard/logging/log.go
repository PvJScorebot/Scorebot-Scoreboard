package logging

import (
	"fmt"
	"io"
	"log"
	"os"
)

const (
	// Trace is the Tracing level, everything is printed to the Log (it might get noisy).
	Trace Level = iota
	// Debug prints slightly less information, but can still be useful for debugging.
	Debug
	// Info is the standard (and default) log Level, prints out handy information messages.
	Info
	// Warning is exactly how it sounds, any event that occurs of notice, but is not major.
	Warning
	// Error is similar to warning, except more serious.
	Error
	// Fatal means the program cannot continue when this event occurs. Normally the program will exit after this.
	Fatal

	logStackDepth uint8 = 3
)

var (
	// LogDefaultOptions is the default bitwize number that is used for new Log structs that are not
	// given an options number when created. This option number may be changed before running to affect
	// runtime functions.
	LogDefaultOptions = log.Ldate | log.Ltime
	// LogConsoleFile is a pointer to the output that all the console Log structs will use. This can be set to any
	// type of stream that can be implemented as a Writer, including NUL.
	LogConsoleFile io.Writer = os.Stdout
)

// Stack is a type of Log that is an alias for an array where each Log
// function will affect each Log instance in the array.
type Stack []Log

// Level is an alias of a byte that repersents the current Log level.
type Level int8

// Log is an interface for any type of struct that supports standard Logging functions.
type Log interface {
	// Info writes a information message to the Log instance.
	Info(string, ...interface{})
	// Error writes a error message to the Log instance.
	Error(string, ...interface{})
	// Fatal writes a fatal message to the Log instance. This function
	// will result in the program exiting with a non-zero error code after being called.
	Fatal(string, ...interface{})
	// Trace writes a tracing message to the Log instance.
	Trace(string, ...interface{})
	// Debug writes a debugging message to the Log instance.
	Debug(string, ...interface{})
	// Warning writes a warning message to the Log instance.
	Warning(string, ...interface{})
}
type fileLog struct {
	logFile string
	streamLog
}
type streamLog struct {
	logLevel  Level
	logWriter *log.Logger
}
type logHandler interface {
	Level() Level
	Writer() *log.Logger
}

// Add appends the specified Log 'l' the Stack array.
func (s *Stack) Add(l Log) {
	if l == nil {
		return
	}
	*s = append(*s, l)
}

// NewConsole returns a console logger that uses the LogConsoleFile writer.
func NewConsole(l Level) Log {
	return NewWriterOptions(l, LogDefaultOptions, LogConsoleFile)
}

// NewStack returns a Stack struct that contains the Log instances
// specified in the 'l' vardict.
func NewStack(l ...Log) *Stack {
	s := Stack(l)
	return &s
}

// String returns the name of the current Level.
func (l *Level) String() string {
	switch *l {
	case Trace:
		return "TRACE"
	case Debug:
		return "DEBUG"
	case Info:
		return " INFO"
	case Warning:
		return " WARN"
	case Error:
		return "ERROR"
	case Fatal:
		return "FATAL"
	}
	return ""
}
func (l *streamLog) Level() Level {
	return l.logLevel
}
func (l *streamLog) Writer() *log.Logger {
	return l.logWriter
}

// NewWriter returns a Log instance based on the Writer 'w' for the logging output.
func NewWriter(l Level, w io.Writer) Log {
	return NewWriterOptions(l, LogDefaultOptions, w)
}

// NewConsoleOptions returns a console logger using the LogConsoleFile file for console ouput and
// allows specifying non-default Logging options.
func NewConsoleOptions(l Level, o int) Log {
	return NewWriterOptions(l, o, LogConsoleFile)
}

// NewFile will attempt to create a File backed Log instance that will write to file 's'.
// This function will truncate the file before starting a new Log. If you need to append to a existing log file.
// use the NewWriter function.
func NewFile(l Level, s string) (Log, error) {
	return NewFileOptions(l, LogDefaultOptions, true, s)
}

// Info writes a information message to the Log instance.
func (s *Stack) Info(m string, v ...interface{}) {
	for i := range *s {
		if b, ok := (*s)[i].(logHandler); ok {
			writeToLog(b.Writer(), b.Level(), Info, logStackDepth+1, m, v)
		} else {
			(*s)[i].Info(m, v...)
		}
	}
}

// Error writes a error message to the Log instance.
func (s *Stack) Error(m string, v ...interface{}) {
	for i := range *s {
		if b, ok := (*s)[i].(logHandler); ok {
			writeToLog(b.Writer(), b.Level(), Error, logStackDepth+1, m, v)
		} else {
			(*s)[i].Error(m, v...)
		}
	}
}

// Fatal writes a fatal message to the Log instance. This function
// will result in the program exiting with a non-zero error code after being called.
func (s *Stack) Fatal(m string, v ...interface{}) {
	for i := range *s {
		if b, ok := (*s)[i].(logHandler); ok {
			writeToLog(b.Writer(), b.Level(), Fatal, logStackDepth+1, m, v)
		} else {
			(*s)[i].Fatal(m, v...)
		}
	}
	os.Exit(1)
}

// Trace writes a tracing message to the Log instance.
func (s *Stack) Trace(m string, v ...interface{}) {
	for i := range *s {
		if b, ok := (*s)[i].(logHandler); ok {
			writeToLog(b.Writer(), b.Level(), Trace, logStackDepth+1, m, v)
		} else {
			(*s)[i].Trace(m, v...)
		}
	}
}

// Debug writes a debugging message to the Log instan
func (s *Stack) Debug(m string, v ...interface{}) {
	for i := range *s {
		if b, ok := (*s)[i].(logHandler); ok {
			writeToLog(b.Writer(), b.Level(), Debug, logStackDepth+1, m, v)
		} else {
			(*s)[i].Debug(m, v...)
		}
	}
}

// Warning writes a warning message to the Log instance.
func (s *Stack) Warning(m string, v ...interface{}) {
	for i := range *s {
		if b, ok := (*s)[i].(logHandler); ok {
			writeToLog(b.Writer(), b.Level(), Warning, logStackDepth+1, m, v)
		} else {
			(*s)[i].Warning(m, v...)
		}
	}
}
func (l *streamLog) Info(m string, v ...interface{}) {
	writeToLog(l.logWriter, l.logLevel, Info, logStackDepth, m, v)
}
func (l *streamLog) Error(m string, v ...interface{}) {
	writeToLog(l.logWriter, l.logLevel, Error, logStackDepth, m, v)
}
func (l *streamLog) Fatal(m string, v ...interface{}) {
	writeToLog(l.logWriter, l.logLevel, Fatal, logStackDepth, m, v)
	os.Exit(1)
}
func (l *streamLog) Trace(m string, v ...interface{}) {
	writeToLog(l.logWriter, l.logLevel, Trace, logStackDepth, m, v)
}
func (l *streamLog) Debug(m string, v ...interface{}) {
	writeToLog(l.logWriter, l.logLevel, Debug, logStackDepth, m, v)
}

// NewWriterOptions returns a Log instance based on the Writer 'w' for the logging output and
// allows specifying non-default Logging options.
func NewWriterOptions(l Level, o int, w io.Writer) Log {
	return &streamLog{logLevel: l, logWriter: log.New(w, "", o)}
}
func (l *streamLog) Warning(m string, v ...interface{}) {
	writeToLog(l.logWriter, l.logLevel, Warning, logStackDepth, m, v)
}

// NewFileOptions will attempt to create a File backed Log instance that will write to file 's'.
// This function will truncate the file before starting a new Log. If you need to append to a existing log file.
// use the NewWriter function. This function allows specifying non-default Logging options.
// If the bool 't' is false, the file will NOT be truncated.
func NewFileOptions(l Level, o int, t bool, s string) (Log, error) {
	i := &fileLog{logFile: s}
	i.logLevel = l
	p := os.O_RDWR | os.O_CREATE
	if t {
		p |= os.O_TRUNC
	}
	w, err := os.OpenFile(s, p, 0644)
	if err != nil {
		return nil, err
	}
	i.logWriter = log.New(w, "", o)
	return i, nil
}
func writeToLog(i *log.Logger, c Level, l Level, d uint8, m string, v []interface{}) {
	if c > l {
		return
	}
	if logStackDepth <= 2 {
		d = logStackDepth
	}
	i.Output(int(d), fmt.Sprintf("%s: %s\n", l.String(), fmt.Sprintf(m, v...)))
}
