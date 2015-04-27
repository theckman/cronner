package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"syscall"
	"time"

	"github.com/PagerDuty/godspeed"
	"github.com/nightlyone/lockfile"
	"github.com/tideland/goas/v3/logger"
)

const intErrCode = 200

func runCommand(cmd *exec.Cmd, gs *godspeed.Godspeed, opts *binArgs) (int, []byte, float64, error) {
	// set up the output buffer for the command
	var b bytes.Buffer

	// comnbine stdout and stderr to the same buffer
	// if we actually plan on using the command output
	// otherwise, /dev/null
	if opts.AllEvents || opts.FailEvent || opts.LogFail {
		cmd.Stdout = &b
		cmd.Stderr = &b
	} else {
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	// build a new lockFile
	lockFile, err := lockfile.New(path.Join(opts.LockDir, fmt.Sprintf("cronner-%v.lock", opts.Label)))

	// make sure we weren't given a bad path
	// and only care if we are doing locking
	if err != nil && opts.Lock {
		logger.Criticalf("failure initializing lockfile: %v", err)
		return intErrCode, nil, 0, err
	}

	// grab the lock
	if opts.Lock {
		if err := lockFile.TryLock(); err != nil {
			logger.Criticalf("failed to obtain lock on '%v': %v", lockFile, err)
			return intErrCode, nil, 0, err
		}
	}

	// get the current (start) time in the UTC epoch
	// and run the command
	s := time.Now().UTC()
	err = cmd.Run()

	// This next section computes the wallclock run time in ms.
	// However, there is the unfortunate limitation in that
	// it uses the clock that gets adjusted by ntpd. Within pure
	// Go, we don't have access to CLOCK_MONOTONIC_RAW.
	//
	// However, based on our usage I don't think we care about it
	// being off by a few milliseconds.
	t := time.Since(s).Seconds() * 1000

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
	if opts.Lock {
		if lockErr := lockFile.Unlock(); lockErr != nil {
			logger.Criticalf("failed to unlock: '%v': %v", lockFile, lockErr)

			// if the command didn't fail, but unlocking did
			// replace the command error with the unlock error
			if err == nil {
				err = lockErr
			}
		}
	}

	// emit the metric for how long it took us and return code
	gs.Timing(fmt.Sprintf("%v.time", opts.Label), t, nil)
	gs.Gauge(fmt.Sprintf("%v.exit_code", opts.Label), float64(ret), nil)

	return ret, b.Bytes(), t, err
}
