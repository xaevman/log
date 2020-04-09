//  ---------------------------------------------------------------------------
//
//  all_test.go
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
    "fmt"
    "os"
    "testing"
    "path/filepath"

    "github.com/xaevman/log"
)

// TestLog creates a new log, writes a few entries to it, rotates the log,
// and then closes it out. At the end of the test two files should exist:
// logs/info.log and logs/old/<date>-info.log
func TestLog(t *testing.T) {
    if err := os.RemoveAll("./logs"); err != nil {
        t.Error(err)
    }

    iLog := New("info", "./logs", BufferedFile)
    eLog := New("error", "./logs", DirectFile)

    iLog.Print(log.NewLogMsg("info", "this is a new INFO log", 2))
    eLog.Print(log.NewLogMsg("error", "this is a new ERROR log", 2))
    iLog.Print(log.NewLogMsg("info", "testing 123", 2))
    eLog.Print(log.NewLogMsg("error", "testing 456", 2))
    iLog.Print(log.NewLogMsg("info", "testing 789", 2))

    iLog = Rotate(iLog)
    eLog = Rotate(eLog)

    iLog.Print(log.NewLogMsg("info", "testing 123 **after** rotate", 2))
    eLog.Print(log.NewLogMsg("error", "testing 456 after rotate", 2))

    iLog.Close()
    eLog.Close()

    oldLogs, err := filepath.Glob("./logs/old/*info.log")
    if err != nil {
        t.Error(err)
    }
    if len(oldLogs) != 1 {
        t.Error(fmt.Errorf("More than one old info.log file found"))
    }

    oldLogs, err = filepath.Glob("./logs/old/*error.log")
    if err != nil {
        t.Error(err)
    }
    if len(oldLogs) != 1 {
        t.Error(fmt.Errorf("More than one old error.log file found"))
    }

    _, err = os.Stat("logs/info.log")
    if err != nil {
        t.Error(err)
    }
    _, err = os.Stat("logs/error.log")
    if err != nil {
        t.Error(err)
    }



}
