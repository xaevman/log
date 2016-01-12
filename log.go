package log

import (
    "container/list"
    "fmt"
    "path/filepath"
    "runtime"
    "strings"
    "sync"
    "time"
)

var timeFormat = "2006/01/02 15:04:05.000000"

// Log represents the interface for a generalized log object.
type LogNotify interface {
    Print(msg string)
}

// LogBuffer holds a specified number of logs in memory for rendering
// by the http logs Uri.
type LogBuffer struct {
    changed bool
    lock    sync.RWMutex
    logs    *list.List
    maxSize int
}

// NewLogBuffer returns a pointer to a new LogBuffer instance.
func NewLogBuffer(maxSize int) *LogBuffer {
    logBuffer := &LogBuffer {
        logs    : list.New(),
        maxSize : maxSize,
    }

    return logBuffer
}

// HasChanged reports whether or not the buffer has been modified
// since it was last read.
func (this *LogBuffer) HasChanged() bool {
    this.lock.RLock()
    defer this.lock.RUnlock()

    return this.changed
}

// Print formats and adds a new message to the log buffer. If the new
// message causes the log buffer to grow larger than its maxSize, it
// truncates the end oldest entry in the buffer. Once the message is
// stored, the changed flag is set to true.
func (this *LogBuffer) Print(msg string) {
    this.lock.Lock()
    defer this.lock.Unlock()

    this.logs.PushFront(msg)

    if this.logs.Len() > this.maxSize {
        this.logs.Remove(this.logs.Back())
    }

    this.changed = true
}

// ReadAll returns a list of all log messages currently in the buffer.
func (this *LogBuffer) ReadAll() []string {
    this.lock.RLock()
    defer this.lock.RUnlock()

    results := make([]string, 0, this.logs.Len())
    for e := this.logs.Front(); e != nil; e = e.Next() {
        results = append(results, e.Value.(string))
    }

    this.changed = false

    return results
}

// SetMaxSize changes the configured maxSize for the buffer, and chops
// the oldest entries off the buffer in the case that it is currently larger
// than the newly configured maxSize.
func (this *LogBuffer) SetMaxSize(maxSize int) {
    this.lock.Lock()
    defer this.lock.Unlock()

    this.maxSize = maxSize

    diff := this.logs.Len() - maxSize
    if diff < 0 {
        for i := 0; i > diff; i-- {
            this.logs.Remove(this.logs.Back())
        }
    }
}

// FormatLogMsg formats a log given the standard logging format
// yyyy/mm/dd MM:HH:SS.ssssss [NAME] <file:line> msg
func FormatLogMsg(name, format string, callDepth int, v ...interface{}) string {
    var msg string
    if len(v) < 1 {
        msg = format
    } else {
        msg = fmt.Sprintf(format, v...)
    }

    _, file, line, ok := runtime.Caller(callDepth)
    if ok {
        file = fmt.Sprintf(" <%s:%d>", filepath.Base(file), line)
    }

    if msg[len(msg) - 1] == '\n' {
        return fmt.Sprintf(
            "%s [%s]%s %s",
            time.Now().Format(timeFormat),
            strings.ToUpper(name),
            file,
            msg,
        )
    }

    return fmt.Sprintf(
        "%s [%s]%s %s\n",
        time.Now().Format(timeFormat),
        strings.ToUpper(name),
        file,
        msg,
    )
}