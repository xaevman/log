//  ---------------------------------------------------------------------------
//
//  DirectLog.go
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

    "log"
    "os"
    "sync"
)

// DirectLog represents a file-backed logger and enforces a standardized
// logging format. New logging entries are written immediately to the 
// backing file.
type DirectLog struct {
    baseDir  string
    enabled  bool
    file     *os.File
    lock     sync.RWMutex
    logger   *log.Logger
    name     string
}

// BaseDir returns the base directory of the file backing this DirectLog instance.
func (this *DirectLog) BaseDir() string {
    this.lock.RLock()
    defer this.lock.RUnlock()

    return this.baseDir
}

// Close disables the DirectLog instance, flushes any remaining entries to disk, and
// then closes the backing log file.
func (this *DirectLog) Close() {
    this.lock.Lock()
    defer this.lock.Unlock()

    this.enabled = false

    this.print(xlog.FormatLogMsg(
        this.name,
        "==== Close log ====",
        3,
    ))

    this.file.Sync()
    this.file.Close()
}

// SetEnabled temporarily enables/disables the DirectLog instance.
func (this *DirectLog) SetEnabled(enabled bool) {
    this.lock.Lock()
    defer this.lock.Unlock()

    this.enabled = enabled
}

// Name returns the friendly name of the log. 
func (this *DirectLog) Name() string {
    this.lock.RLock()
    defer this.lock.RUnlock()

    return this.name
}

// Print formats and buffers a new log entry as long as the DirectLog instance
// is enabled.
func (this *DirectLog) Print(msg string) {
    this.lock.RLock()
    defer this.lock.RUnlock()

    if !this.enabled {
        return
    }

    this.print(msg)
}

func (this *DirectLog) print(msg string) {
    log.Print(msg)
    this.logger.Print(msg)
}
