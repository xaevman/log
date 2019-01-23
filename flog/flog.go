//  ---------------------------------------------------------------------------
//
//  init.go
//
//  Copyright (c) 2014, Jared Chavez.
//  All rights reserved.
//
//  Use of this source code is governed by a BSD-style
//  license that can be found in the LICENSE file.
//
//  -----------

// Package flog provides facilities for using and managing
// file-backed logger objects.
package flog

import (
	"github.com/xaevman/crash"
	xlog "github.com/xaevman/log"
	"github.com/xaevman/shutdown"

	"fmt"
	"log"
	"os"
	"path"
	"time"
)

// Default flush interval, in seconds, for BufferedLog instances.
const DefaultFlushIntervalSec = 5

// Log file open flags.
const FLogOpenFlags = os.O_RDWR | os.O_APPEND | os.O_CREATE

// Enumeration of different FLog implementations.
const (
	BufferedFile = iota
	DirectFile
)

// FLog provides a common interface for different file-backed logs. This package
// includes two primary implementations; BufferedLog and DirectLog.
type FLog interface {
	BaseDir() string
	Close()
	SetEnabled(bool)
	Name() string
	Print(msg *xlog.LogMsg)
}

// New returns a new FLog instance of the requested type. The backing log file is
// created or opened for append.
func New(name, logPath string, logType int) FLog {
	var newLog FLog

	mkdir(logPath)

	f, err := os.OpenFile(
		path.Join(logPath, name+".log"),
		FLogOpenFlags,
		0660,
	)
	if err != nil {
		return nil
	}

	switch logType {
	case BufferedFile:

		bLog := BufferedLog{
			baseDir:   logPath,
			shutdown:  shutdown.New(),
			enabled:   true,
			flushSec:  DefaultFlushIntervalSec,
			flushChan: make(chan interface{}, 0),
			name:      name,
		}

		bLog.file = f

		l := log.New(&bLog.buffer, "", 0)
		bLog.logger = l

		go func() {
			defer crash.HandleAll()
			bLog.asyncFlush()
		}()

		newLog = &bLog
		break

	case DirectFile:

		dLog := DirectLog{
			baseDir: logPath,
			enabled: true,
			name:    name,
		}

		dLog.file = f

		l := log.New(dLog.file, "", 0)
		dLog.logger = l

		newLog = &dLog
		break
	}

	initMsg := xlog.NewLogMsg(
		name,
		"==== Log init ====",
		2,
	)

	newLog.Print(initMsg)

	return newLog
}

// Rotate takes a given FLog instance, closes it, timestamps and moves the
// backing log file into an old subdirectory, before opening and returning a new
// FLog instance at the original location.
func Rotate(log FLog) FLog {
	log.Close()

	mkPath := path.Join(log.BaseDir(), "old")

	mkdir(mkPath)

	now := time.Now()
	newPath := path.Join(
		mkPath,
		fmt.Sprintf(
			"%d%d%d-%s.log",
			now.Year(),
			now.Month(),
			now.Day(),
			log.Name(),
		),
	)
	oldPath := path.Join(
		log.BaseDir(),
		log.Name()+".log",
	)

	err := os.Rename(
		oldPath,
		newPath,
	)

	if err != nil {
		panic(err)
	}

	var newLog FLog
	bLog, ok := log.(*BufferedLog)

	if ok {
		newLog = New(log.Name(), log.BaseDir(), BufferedFile)
		newLog.(*BufferedLog).SetFlushIntervalSec(bLog.FlushIntervalSec())
	} else {
		newLog = New(log.Name(), log.BaseDir(), DirectFile)
	}

	return newLog
}

// init sets the default Logger flags to match the FLog packages preferred flags.
func init() {
	log.SetFlags(0)
}

// mkdir wraps os.MkdirAll with default privs of 770 and panics on errors.
func mkdir(path string) {
	err := os.MkdirAll(path, 0770)
	if err != nil {
		panic(err)
	}
}
