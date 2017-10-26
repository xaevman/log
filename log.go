//  ---------------------------------------------------------------------------
//
//  log.go
//
//  Copyright (c) 2015, Jared Chavez.
//  All rights reserved.
//
//  Use of this source code is governed by a BSD-style
//  license that can be found in the LICENSE file.
//
//  -----------

// Package log provides interfaces, basic types and formatting helpers
// for common application logging tasks.
package log

import (
	"bytes"
	"container/ring"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var timeFormat = "2006/01/02 15:04:05.0000"

type DebugLogger interface {
	Debug(format string, v ...interface{})
}

type ErrorLogger interface {
	Error(format string, v ...interface{})
}

type InfoLogger interface {
	Info(format string, v ...interface{})
}

type LogMsg struct {
	Timestamp time.Time
	Name      string
	File      string
	Line      int
	Message   string
}

func (this *LogMsg) String() string {
	var buffer bytes.Buffer

	if this.File != "" {
		buffer.WriteString(" <")
		buffer.WriteString(this.File)
		if this.Line != 0 {
			buffer.WriteString(fmt.Sprintf(":%d", this.Line))
		}
		buffer.WriteString("> ")
	}

	if this.Message[len(this.Message)-1] == '\n' {
		return fmt.Sprintf(
			"%s [%s]%s %s",
			this.Timestamp.Format(timeFormat),
			strings.ToUpper(this.Name),
			buffer.String(),
			this.Message,
		)
	}

	return fmt.Sprintf(
		"%s [%s]%s %s\n",
		this.Timestamp.Format(timeFormat),
		strings.ToUpper(this.Name),
		buffer.String(),
		this.Message,
	)
}

// LogCloser represents the interface for a closable log object, such
// as those provided by file-backed logging.
type LogCloser interface {
	Close()
}

// Log represents the interface for a generalized log object.
type LogNotify interface {
	Print(msg *LogMsg)
}

// LogToggler represents the interface needed to temporarily enabled/disable
// a given logger.
type LogToggler interface {
	SetEnabled(bool)
}

// LogBuffer holds a specified number of logs in memory for rendering
// by the http logs Uri.
type LogBuffer struct {
	changed bool
	enabled bool
	lock    sync.RWMutex
	logs    *ring.Ring
}

// NewLogBuffer returns a pointer to a new LogBuffer instance.
func NewLogBuffer(maxSize int) *LogBuffer {
	logBuffer := &LogBuffer{
		enabled: true,
		logs:    ring.New(maxSize),
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
func (this *LogBuffer) Print(msg *LogMsg) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if !this.enabled {
		return
	}

	this.logs.Value = msg
	this.logs = this.logs.Next()

	this.changed = true
}

// ReadAll returns a list of all log messages currently in the buffer.
func (this *LogBuffer) ReadAll() []*LogMsg {
	this.lock.RLock()
	defer this.lock.RUnlock()

	results := make([]*LogMsg, 0, this.logs.Len())
	this.logs.Do(func(item interface{}) {
		if item == nil {
			return
		}

		results = append(results, item.(*LogMsg))
	})

	this.changed = false

	return results
}

// MarshalJSON implements the standard go json marshaler for the
// LogBuffer type.
func (this *LogBuffer) MarshalJSON() ([]byte, error) {
	this.lock.RLock()
	defer this.lock.RUnlock()

	results := make([]*LogMsg, 0, this.logs.Len())
	this.logs.Do(func(item interface{}) {
		if item == nil {
			return
		}

		results = append(results, item.(*LogMsg))
	})

	return json.Marshal(&results)
}

// SetEnabled temporarily enables/disables the logger.
func (this *LogBuffer) SetEnabled(enabled bool) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.enabled = enabled
}

// NewLogMsg generates a log object given the standard logging format
// yyyy/mm/dd MM:HH:SS.ssssss [NAME] <file:line> msg
func NewLogMsg(name, format string, callDepth int, v ...interface{}) *LogMsg {
	var msg string
	if len(v) < 1 {
		msg = format
	} else {
		msg = fmt.Sprintf(format, v...)
	}

	newLog := &LogMsg{
		Timestamp: time.Now(),
		Name:      name,
		Message:   msg,
	}

	_, file, line, ok := runtime.Caller(callDepth)
	if ok {
		newLog.File = filepath.Base(file)
		newLog.Line = line
	}

	return newLog
}
