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

// execCmd is a function to run a command and send
// the error value back through a channel
func execCmd(cmd *exec.Cmd, c chan<- error) {
	c <- cmd.Run()
	close(c)
}

// runCommand is a function that handles the entire process of running a command:
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
// * ([]byte) command output
// * (float64) run time
// * (error) WISOTT
func runCommand(cmd *exec.Cmd, gs *godspeed.Godspeed, opts *binArgs, host, uuidStr string) (int, []byte, float64, error) {
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

	var s time.Time
	ch := make(chan error)

	// if we have a timer value, do all the extra logic to
	// use the ticker to send out warning events
	//
	// otherwise, KISS
	if opts.WarnAfter > 0 {
		// use time.Tick() instead of time.NewTicker() because
		// we don't ever need to run Stop() on this ticker as cronner
		// won't live much beyond the command returning
		tickChan := time.Tick(time.Second * time.Duration(opts.WarnAfter))

		// get the current (start) time since the UTC epoch
		// and run the command
		s = time.Now().UTC()
		go execCmd(cmd, ch)

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
					title := fmt.Sprintf("Cron %v still running after %d seconds on %v", opts.Label, int64(runSecs), host)
					body := fmt.Sprintf("UUID: %v\nrunning for %v seconds", uuidStr, int64(runSecs))
					emitEvent(title, body, opts.Label, "warning", uuidStr, gs)
				}
			}
		}
	} else {
		// get the current (start) time since the UTC epoch
		// and run the command
		s = time.Now().UTC()
		go execCmd(cmd, ch)
		err = <-ch
	}

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
