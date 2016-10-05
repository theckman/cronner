// Copyright 2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"syscall"
	"time"

	"github.com/theckman/go-flock"
	"github.com/tideland/golib/logger"
)

const intErrCode = 200

// MaxBody is the maximum length of a event body
const MaxBody = 4096

// execCmd is a function to run a command and send
// the error value back through a channel
func execCmd(cmd *exec.Cmd, c chan<- error) {
	c <- cmd.Run()
	close(c)
}

// handleCommand is a function that handles the entire process of running a command:
//
// * file-based locking for the command
// * actually running the command
// * timing how long it takes and emitting a metric for it
// * tracking command return codes and emitting a metric for it
// * emitting warning metrics if a command has exceeded its running time
//
// it returns the following:
//
// * (int) return code
// * (float64) run time
func handleCommand(hndlr *cmdHandler) (int, []byte, float64, error) {
	if hndlr.opts.AllEvents {
		// emit a DD event to indicate we are starting the job
		emitEvent(fmt.Sprintf("Cron %v starting on %v", hndlr.opts.Label, hndlr.hostname), fmt.Sprintf("UUID: %v\n", hndlr.uuid), hndlr.opts.Label, "info", hndlr)
	}

	// set up the output buffers for the command
	var b bytes.Buffer

	// setup multiple streams only on passthru
	// combine stdout and stderr to the same buffer
	// if we actually plan on using the command output
	// otherwise, /dev/null
	if hndlr.opts.AllEvents || hndlr.opts.FailEvent || hndlr.opts.LogFail {
		if hndlr.opts.Passthru {
			hndlr.cmd.Stdout = io.MultiWriter(os.Stdout, &b)
			hndlr.cmd.Stderr = io.MultiWriter(os.Stderr, &b)
		} else {
			hndlr.cmd.Stdout = &b
			hndlr.cmd.Stderr = &b
		}
	} else {
		if hndlr.opts.Passthru {
			hndlr.cmd.Stdout = os.Stdout
			hndlr.cmd.Stderr = os.Stderr
		} else {
			hndlr.cmd.Stdout = nil
			hndlr.cmd.Stderr = nil
		}
	}

	// build a new lockFile
	lockFile := flock.NewFlock(path.Join(hndlr.opts.LockDir, fmt.Sprintf("cronner-%v.lock", hndlr.opts.Label)))

	var err error

	// grab the lock
	if hndlr.opts.Lock {
		locked, err := lockFile.TryLock()

		if err != nil {
			retErr := fmt.Errorf("failed to obtain lock on '%v': %v", lockFile, err)
			return intErrCode, nil, -1, retErr
		}

		if !locked && hndlr.opts.WaitSeconds == 0 {
			retErr := fmt.Errorf("failed to obtain lock on '%v': locked by another process", lockFile)
			return intErrCode, nil, -1, retErr
		} else if !locked && hndlr.opts.WaitSeconds > 0 {
			tick := time.NewTicker(time.Second * time.Duration(hndlr.opts.WaitSeconds))

		GotLock:
			for {
				select {
				case _ = <-tick.C:
					retErr := fmt.Errorf("timeout exceeded (%ds) waiting for the file lock", hndlr.opts.WaitSeconds)
					return intErrCode, nil, -1, retErr
				default:
					locked, err = lockFile.TryLock()

					if !locked || err != nil {
						time.Sleep(time.Second * 1)
						continue
					}

					break GotLock
				}
			}
		}
	}

	var s time.Time
	ch := make(chan error)

	// if we have a timer value, do all the extra logic to
	// use the ticker to send out warning events
	//
	// otherwise, KISS
	if hndlr.opts.WarnAfter > 0 {
		// use time.Tick() instead of time.NewTicker() because
		// we don't ever need to run Stop() on this ticker as cronner
		// won't live much beyond the command returning
		tickChan := time.Tick(time.Second * time.Duration(hndlr.opts.WarnAfter))

		// get the current (start) time since the UTC epoch
		// and run the command
		s = time.Now().UTC()
		go execCmd(hndlr.cmd, ch)

		// this is an open loop to wait for either the command to return
		// or time to be sent over the ticker channel
		//
		// the WaitLoop label is used to break from the select statement
	WaitLoop:
		for {
			// wait for either the command channel to return an error value
			// or wait for the ticket channel to return a time.Time value
			select {
			case m := <-ch:
				// the comand returned; set the error vailue and bail out of here
				err = m
				break WaitLoop
			case _, ok := <-tickChan:
				if ok {
					runSecs := time.Since(s).Seconds()
					title := fmt.Sprintf("Cron %v still running after %d seconds on %v", hndlr.opts.Label, int64(runSecs), hndlr.hostname)
					body := fmt.Sprintf("UUID: %v\nrunning for %v seconds", hndlr.uuid, int64(runSecs))
					emitEvent(title, body, hndlr.opts.Label, "warning", hndlr)
				}
			}
		}
	} else {
		// get the current (start) time since the UTC epoch
		// and run the command
		s = time.Now().UTC()
		go execCmd(hndlr.cmd, ch)
		err = <-ch
	}

	// This next section computes the wallclock run time in ms.
	// However, there is the unfortunate limitation in that
	// it uses the clock that gets adjusted by ntpd. Within pure
	// Go, we don't have access to CLOCK_MONOTONIC_RAW.
	//
	// However, based on our usage I don't think we care about it
	// being off by a few milliseconds.
	wallRtMs := time.Since(s).Seconds() * 1000

	// calculate the return code of the command
	// default to return code 0: success
	//
	// this is being done within the lock because
	// even if we fail to remove the lockfile, we still
	// need to know what the command did.
	var ret int
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			status := ee.Sys().(syscall.WaitStatus)
			ret = status.ExitStatus()
		} else {
			ret = intErrCode
		}
	}

	// unlock
	if hndlr.opts.Lock {
		if lockErr := lockFile.Unlock(); lockErr != nil {
			// if the command didn't fail, but unlocking did
			// replace the command error with the unlock error
			// otherwise just print the error
			retErr := fmt.Errorf("failed to unlock: '%v': %v", lockFile, lockErr)
			if err == nil {
				err = retErr
			} else {
				logger.Errorf(retErr.Error())
			}
		}
	}

	// emit the metric for how long it took us and return code
	hndlr.gs.Timing(fmt.Sprintf("%v.time", hndlr.opts.Label), wallRtMs, nil)
	hndlr.gs.Gauge(fmt.Sprintf("%v.exit_code", hndlr.opts.Label), float64(ret), nil)

	out := b.Bytes()

	// default variables are for success
	// we change them later if there was a failure
	msg := "succeeded"
	alertType := "success"

	// if the command failed change the state variables to their failure values
	if err != nil {
		msg = "failed"
		alertType = "error"
	}

	if hndlr.opts.AllEvents || (hndlr.opts.FailEvent && alertType == "error") {
		// build the pieces of the completion event
		title := fmt.Sprintf("Cron %v %v in %.5f seconds on %v", hndlr.opts.Label, msg, wallRtMs/1000, hndlr.hostname)

		body := fmt.Sprintf("UUID: %v\nexit code: %d\n", hndlr.uuid, ret)
		if err != nil {
			er := regexp.MustCompile("^exit status ([-]?\\d)")

			// do not show the 'more:' line, if the line is just telling us
			// what the exit code is
			if !er.MatchString(err.Error()) {
				body = fmt.Sprintf("%vmore: %v\n", body, err.Error())
			}
		}

		var cmdOutput string

		if len(out) > 0 {
			cmdOutput = string(out)
		} else {
			cmdOutput = "(none)"
		}

		body = fmt.Sprintf("%voutput: %v", body, cmdOutput)

		emitEvent(title, body, hndlr.opts.Label, alertType, hndlr)
	}

	// DRY: stdout/stderr has already been printed
	if hndlr.opts.Passthru {
		hndlr.opts.Sensitive = true
	}

	// this code block is meant to be ran last
	if alertType == "error" && hndlr.opts.LogFail {
		filename := path.Join(hndlr.opts.LogPath, fmt.Sprintf("%v-%v.out", hndlr.opts.Label, hndlr.uuid))
		if !writeOutput(filename, out, hndlr.opts.Sensitive) {
			os.Exit(1)
		}
	}

	return ret, out, wallRtMs, err
}

// emit a godspeed (dogstatsd) event
func emitEvent(title, body, label, alertType string, hndlr *cmdHandler) {
	var buf bytes.Buffer

	// if the event's body is bigger than MaxBody
	if len(body) > MaxBody {
		// push the first MaxBody/2 bytes in to the buffer
		buf.WriteString(body[0 : MaxBody/2])

		// add indication of truncated output to the buffer
		buf.WriteString("...\n=== OUTPUT TRUNCATED ===\n")

		// add the last 1024 bytes to the buffer
		buf.WriteString(body[len(body)-((MaxBody/2)+1) : len(body)-1])

		body = string(buf.Bytes())
	}

	fields := make(map[string]string)
	fields["source_type_name"] = "cronner"

	if len(alertType) > 0 {
		fields["alert_type"] = alertType
	}

	if len(hndlr.uuid) > 0 {
		fields["aggregation_key"] = hndlr.uuid
	}

	tags := []string{"source_type:cronner", fmt.Sprintf("cronner_label_name:%v", label)}

	if len(hndlr.opts.EventGroup) > 0 {
		tags = append(tags, fmt.Sprintf("cronner_group:%v", hndlr.opts.EventGroup))
	}

	hndlr.gs.Event(title, body, fields, tags)
}

// bailOut is for failures during logfile writing
func bailOut(out []byte, sensitive bool) bool {
	if !sensitive {
		fmt.Fprintf(os.Stderr, "here is the output in hopes you are looking here:\n\n%v", string(out))
		os.Exit(1)
	}
	return false
}

// writeOutput saves the output (out) to the file specified
func writeOutput(filename string, out []byte, sensitive bool) bool {
	// check to see whehter or not the output file already exists
	// this should really never happen, but just in case it does...
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "flagrant error: output file '%v' already exists\n", filename)
		return bailOut(out, sensitive)
	}

	outFile, err := os.Create(filename)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening file to save command output: %v\n", err.Error())
		return bailOut(out, sensitive)
	}

	defer outFile.Close()

	if err = outFile.Chmod(0400); err != nil {
		fmt.Fprintf(os.Stderr, "error setting permissions (0400) on file '%v': %v\n", filename, err.Error())
		return bailOut(out, sensitive)
	}

	nwrt, err := outFile.Write(out)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing to file '%v': %v\n", filename, err.Error())
		return bailOut(out, sensitive)
	}

	if nwrt != len(out) {
		fmt.Fprintf(os.Stderr, "error writing to file '%v': number of bytes written not equal to output (total: %d, written: %d)\n", filename, len(out), nwrt)
		return bailOut(out, sensitive)
	}

	return true
}
