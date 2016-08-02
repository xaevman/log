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
    xlog "github.com/xaevman/log"

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
    baseDir  string
    buffer   bytes.Buffer
    chClose  chan interface{}
    enabled  bool
    file     *os.File
    flushSec int32
    lock     sync.RWMutex
    logger   *log.Logger
    name     string
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
    this.lock.Lock()
    this.enabled = false
    this.lock.Unlock()

    this.print(xlog.NewLogMsg(this.name, "==== Close log ====", 2))

    // stop flush routine
    this.chClose <- nil

    // flush logs
    this.flushLogs()

    // close file
    this.file.Close()
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
    run := true

    for run {
        flushSec := atomic.LoadInt32(&this.flushSec)

        select {
        case <-this.chClose:
            run = false
            this.print(xlog.NewLogMsg(
                this.name,
                "Async log shutdown",
                3,
            ))
            continue
        case <-time.After(time.Duration(flushSec) * time.Second):
            this.flushLogs()
        }
    }

    this.chClose <- nil
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
}

func (this *BufferedLog) print(msg *xlog.LogMsg) {
    this.lock.Lock()
    defer this.lock.Unlock()

    log.Print(msg)
    this.logger.Print(msg)
}
