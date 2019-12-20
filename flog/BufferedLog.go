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
	"github.com/xaevman/shutdown"

	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/trace"
	"sync"
	"sync/atomic"
	"time"
)

const blMaxBufferSize = 64 * 1024 // 64KB

// BufferedLog represents a buffered, file-backed logger and enforces a standardized
// logging format. New logging entries are sent to a memory buffer and
// periodically flushed to the backing file at configurable intervals
// by a seperate goroutine.
type BufferedLog struct {
	baseDir   string
	buffer    bytes.Buffer
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
	ctx := context.Background()

	trace.WithRegion(ctx, "BaseDir().acquireReadLock", this.lock.RLock)
	defer this.lock.RUnlock()

	return this.baseDir
}

// BufferCap returns the current capacity of the underlying memory buffer.
func (this *BufferedLog) BufferCap() int {
	ctx := context.Background()

	trace.WithRegion(ctx, "BufferCap().acquireReadLock", this.lock.RLock)
	defer this.lock.RUnlock()

	return this.buffer.Cap()
}

// Close disables the BufferedLog instance, flushes any remaining entries to disk, and
// then closes the backing log file.
func (this *BufferedLog) Close() {
	if this.hasClosed {
		return
	}

	this.hasClosed = true

	ctx := context.Background()
	this.print(ctx, xlog.NewLogMsg(this.name, "==== Close log ====", 2))

	// stop flush routine
	this.shutdown.Start()

	if this.shutdown.WaitForTimeout() {
		this.print(ctx, xlog.NewLogMsg(this.name, "Timeout waiting on shutdown", 2))
	}

	// flush logs
	this.flushLogs()

	// close file
	this.file.Close()
}

// SetEnabled temporarily enables/disables the log instance.
func (this *BufferedLog) SetEnabled(enabled bool) {
	ctx := context.Background()

	trace.WithRegion(ctx, "SetEnabled().acquireLock", this.lock.Lock)
	defer this.lock.Unlock()

	this.enabled = enabled
}

// FlushInterval returns the interval between log flushes in seconds.
func (this *BufferedLog) FlushIntervalSec() int32 {
	return atomic.LoadInt32(&this.flushSec)
}

// Name returns the friendly name of the log.
func (this *BufferedLog) Name() string {
	ctx := context.Background()

	trace.WithRegion(ctx, "Name().acquireReadLock", this.lock.RLock)
	defer this.lock.RUnlock()

	return this.name
}

// Print formats and buffers a new log entry as long as the BufferedLog instance
// is enabled.
func (this *BufferedLog) Print(msg *xlog.LogMsg) {
	ctx := context.Background()

	trace.WithRegion(ctx, "Print().acquireReadLock", this.lock.RLock)
	if !this.enabled {
		this.lock.RUnlock()
		return
	}
	this.lock.RUnlock()

	this.print(ctx, msg)
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

	ctx := context.Background()

	for {
		flushSec := atomic.LoadInt32(&this.flushSec)

		select {
		case <-this.shutdown.Signal:
			this.print(ctx, xlog.NewLogMsg(
				this.name,
				"Async log shutdown",
				3,
			))
			return
		case <-this.flushChan:
			trace.Log(ctx, "flushChan triggered", "")
			trace.WithRegion(ctx, "flushLogs()", this.flushLogs)
		case <-time.After(time.Duration(flushSec) * time.Second):
			trace.Log(ctx, "flushSecElapsed", fmt.Sprintf("%d", flushSec))
			this.flushLogs()
		}
	}
}

// flushLogs copies the contents of the log buffer into the open log file.
func (this *BufferedLog) flushLogs() {
	ctx := context.Background()

	trace.WithRegion(ctx, "flushLogs().acquireLock", this.lock.Lock)
	defer this.lock.Unlock()

	// flush may have just happened, so check
	// the buffer len again before blocking on the
	// disk
	if this.buffer.Len() < blMaxBufferSize {
		return
	}

	r := trace.StartRegion(ctx, "flushLogs().write")
	_, err := io.Copy(this.file, &this.buffer)
	r.End()
	if err != nil {
		panic(err)
	}

	r = trace.StartRegion(ctx, "flushLogs().sync")
	err = this.file.Sync()
	r.End()
	if err != nil {
		panic(err)
	}
}

func (this *BufferedLog) print(ctx context.Context, msg *xlog.LogMsg) {
	trace.WithRegion(ctx, "print().acquireLock", this.lock.Lock)

	r := trace.StartRegion(ctx, "print().delegates")
	log.Print(msg)
	this.logger.Print(msg)
	r.End()

	if this.buffer.Len() > blMaxBufferSize {
		this.lock.Unlock()

		r = trace.StartRegion(ctx, "print().requestFlush")
		select {
		case this.flushChan <- nil:
			return
		case <-time.After(1 * time.Second):
			return
		}
		r.End()
	} else {
		this.lock.Unlock()
	}
}
