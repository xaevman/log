//  ---------------------------------------------------------------------------
//
//  BufferedLog.go
//
//  Copyright (c) 2014, Jared Chavez.
//  All rights reserved.
//
//  Use of this source code is governed by a BSD-style
//  license that can be found in the LICENSE file.
//
//  -----------

package flog

import (
	"github.com/xaevman/crash"
	xlog "github.com/xaevman/log"
	"github.com/xaevman/shutdown"

	"bytes"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// BufferedLog represents a buffered, file-backed logger and enforces a standardized
// logging format. New logging entries are sent to a memory buffer and
// periodically flushed to the backing file at configurable intervals
// by a seperate goroutine.
type BufferedLog struct {
	baseDir   string
	buffer    bytes.Buffer
	count     uint
	shutdown  *shutdown.Sync
	enabled   bool
	file      *os.File
	flushSec  int32
	flushChan chan interface{}
	hasClosed bool
	lock      sync.RWMutex
	logger    *log.Logger
	name      string
}

// BaseDir returns the base directory of the file backing this BufferedLog instance.
func (this *BufferedLog) BaseDir() string {
	this.lock.RLock()
	defer this.lock.RUnlock()

	return this.baseDir
}

// Close disables the BufferedLog instance, flushes any remaining entries to disk, and
// then closes the backing log file.
func (this *BufferedLog) Close() {
	if this.hasClosed {
		return
	}

	this.hasClosed = true

	this.print(xlog.NewLogMsg(this.name, "==== Close log ====", 2))

	// stop flush routine
	this.shutdown.Start()

	// flush logs
	this.flushLogs()

	// close file
	this.file.Close()

	if this.shutdown.WaitForTimeout() {
		this.print(xlog.NewLogMsg(this.name, "Timeout waiting on shutdown", 2))
	}
}

// SetEnabled temporarily enables/disables the log instance.
func (this *BufferedLog) SetEnabled(enabled bool) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.enabled = enabled
}

// FlushInterval returns the interval between log flushes in seconds.
func (this *BufferedLog) FlushIntervalSec() int32 {
	return atomic.LoadInt32(&this.flushSec)
}

// Name returns the friendly name of the log.
func (this *BufferedLog) Name() string {
	this.lock.RLock()
	defer this.lock.RUnlock()

	return this.name
}

// Print formats and buffers a new log entry as long as the BufferedLog instance
// is enabled.
func (this *BufferedLog) Print(msg *xlog.LogMsg) {
	this.lock.RLock()
	if !this.enabled {
		this.lock.RUnlock()
		return
	}
	this.lock.RUnlock()

	this.print(msg)
}

// SetFlushIntervalSec sets the interval at which the log buffer worker
// will attempt to flush new entries into the backing log file.
func (this *BufferedLog) SetFlushIntervalSec(interval int32) {
	atomic.StoreInt32(&this.flushSec, interval)
}

// asyncFlush is run in a separate goroutine and periodically flushes
// buffered entries to the backing file.
func (this *BufferedLog) asyncFlush() {
	defer this.shutdown.Complete()

	for {
		flushSec := atomic.LoadInt32(&this.flushSec)

		select {
		case <-this.shutdown.Signal:
			this.print(xlog.NewLogMsg(
				this.name,
				"Async log shutdown",
				3,
			))
			return
		case <-this.flushChan:
			this.flushLogs()
		case <-time.After(time.Duration(flushSec) * time.Second):
			this.flushLogs()
		}
	}
}

// flushLogs copies the contents of the log buffer into the open log file.
func (this *BufferedLog) flushLogs() {
	this.lock.Lock()
	defer this.lock.Unlock()

	_, err := io.Copy(this.file, &this.buffer)
	if err != nil {
		panic(err)
	}

	err = this.file.Sync()
	if err != nil {
		panic(err)
	}

	this.count = 0
}

func (this *BufferedLog) print(msg *xlog.LogMsg) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.count++
	log.Print(msg)
	this.logger.Print(msg)

	if this.count > 100 {
		go func() {
			defer crash.HandleAll()
			select {
			case this.flushChan <- nil:
				return
			case <-time.After(5 * time.Second):
				return
			}
		}()
	}
}
