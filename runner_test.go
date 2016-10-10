// Copyright 2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"github.com/theckman/go-flock"

	. "gopkg.in/check.v1"
)

func (t *TestSuite) Test_handleCommand(c *C) {
	//
	// Test a command that finishes in 0.3 seconds
	//
	t.h.cmd = exec.Command("/usr/bin/time", "-p", "/bin/sleep", "0.3")

	retCode, r, runTime, err := handleCommand(t.h)
	c.Assert(err, IsNil)
	c.Check(retCode, Equals, 0)

	stat, ok := <-t.out
	c.Assert(ok, Equals, true)

	timeStatRegex := regexp.MustCompile("^cronner.testCmd.time:([0-9\\.]+)\\|ms$")
	match := timeStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	statFloat, err := strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Check(statFloat, Equals, runTime)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)

	retStatRegex := regexp.MustCompile("^cronner.testCmd.exit_code:([0-9\\.]+)\\|g$")
	match = retStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	retFloat, err := strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Check(retFloat, Equals, float64(0))

	var timely bool

	// assume the command run time will be within 20ms of correct,
	// note sure how tight we can make this window without incurring
	// false-failures.
	if runTime > 300 && runTime < 320 {
		timely = true
	}
	c.Assert(timely, Equals, true)

	timeRegex := regexp.MustCompile("((?m)^real[[:space:]]+([0-9\\.]+)$)")
	match = timeRegex.FindAllStringSubmatch(string(r), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 3)
	c.Check(match[0][2], Equals, "0.30")

	//
	// Test a command that finishes in 1 second
	//

	// Reset variables used
	r = nil
	err = nil
	runTime = 0
	match = nil
	timely = false
	retCode = -512
	timeRegex = nil

	t.h.cmd = exec.Command("/usr/bin/time", "-p", "/bin/sleep", "1")

	retCode, r, runTime, err = handleCommand(t.h)
	c.Assert(err, IsNil)
	c.Check(retCode, Equals, 0)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)

	match = timeStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	statFloat, err = strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Check(statFloat, Equals, runTime)

	if runTime > 1000 && runTime < 1020 {
		timely = true
	}
	c.Check(timely, Equals, true)

	timeRegex = regexp.MustCompile("((?m)^real[[:space:]]+([0-9\\.]+)$)")
	match = timeRegex.FindAllStringSubmatch(string(r), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 3)
	c.Check(match[0][2], Equals, "1.00")

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)

	match = retStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	retFloat, err = strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Check(retFloat, Equals, float64(0))

	//
	// Test a valid return code is given
	//

	// Reset variables used
	r = nil
	err = nil
	runTime = 0
	match = nil
	retCode = -512

	switch runtime.GOOS {
	case "linux":
		t.h.cmd = exec.Command("/bin/false")
	case "darwin":
		t.h.cmd = exec.Command("/usr/bin/false")
	}

	retCode, r, runTime, err = handleCommand(t.h)
	c.Assert(err, Not(IsNil))
	c.Check(retCode, Equals, 1)

	_, ok = <-t.out
	c.Assert(ok, Equals, true)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)

	match = retStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	retFloat, err = strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Check(retFloat, Equals, float64(1))

	//
	// Test that DD events work
	//

	// Reset variables used
	r = nil
	err = nil
	runTime = 0
	match = nil
	retCode = -512

	t.h.cmd = exec.Command("/bin/echo", "somevalue")
	t.h.opts.AllEvents = true

	retCode, r, runTime, err = handleCommand(t.h)
	c.Assert(err, IsNil)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	c.Check(
		string(stat),
		Equals,
		fmt.Sprintf(`_e{35,44}:Cron testCmd starting on brainbox01|UUID: %v\n|k:%v|s:cronner|t:info|#source_type:cronner,cronner_label_name:testCmd`, t.h.uuid, t.h.uuid),
	)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	match = timeStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)
	c.Check(strconv.FormatFloat(runTime, 'f', -1, 64), Equals, match[0][1])

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	c.Check(string(stat), Equals, "cronner.testCmd.exit_code:0|g")

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	c.Check(
		string(stat),
		Equals,
		fmt.Sprintf(`_e{55,77}:Cron testCmd succeeded in %.5f seconds on brainbox01|UUID: %v\nexit code: 0\noutput: somevalue\n|k:%v|s:cronner|t:success|#source_type:cronner,cronner_label_name:testCmd`, runTime/1000, t.h.uuid, t.h.uuid),
	)

	//
	// Test that DD events contain the cronner_group tag
	//

	// Reset variables used
	r = nil
	err = nil
	runTime = 0
	match = nil

	t.h.cmd = exec.Command("/bin/echo", "somevalue")
	t.h.opts.EventGroup = "testgroup"

	_, r, runTime, err = handleCommand(t.h)
	c.Assert(err, IsNil)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	c.Check(
		string(stat),
		Equals,
		fmt.Sprintf(`_e{35,44}:Cron testCmd starting on brainbox01|UUID: %v\n|k:%v|s:cronner|t:info|#source_type:cronner,cronner_label_name:testCmd,cronner_group:testgroup`, t.h.uuid, t.h.uuid),
	)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	match = timeStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)
	c.Check(strconv.FormatFloat(runTime, 'f', -1, 64), Equals, match[0][1])

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	c.Check(string(stat), Equals, "cronner.testCmd.exit_code:0|g")

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	c.Check(
		string(stat),
		Equals,
		fmt.Sprintf(`_e{55,77}:Cron testCmd succeeded in %.5f seconds on brainbox01|UUID: %v\nexit code: 0\noutput: somevalue\n|k:%v|s:cronner|t:success|#source_type:cronner,cronner_label_name:testCmd,cronner_group:testgroup`, runTime/1000, t.h.uuid, t.h.uuid),
	)

	//
	// Test that no output is given
	//

	// Reset variables used
	r = nil
	err = nil
	runTime = 0
	match = nil

	t.h.cmd = exec.Command("/bin/echo", "something")
	t.h.opts.EventGroup = ""

	t.h.opts.LogFail = false
	t.h.opts.Lock = true
	t.h.opts.AllEvents = false

	retCode, r, _, err = handleCommand(t.h)
	c.Assert(err, IsNil)
	c.Check(retCode, Equals, 0)
	c.Check(len(r), Equals, 0)

	// clear the statsd return channel
	_, ok = <-t.out
	c.Assert(ok, Equals, true)
	_, ok = <-t.out
	c.Assert(ok, Equals, true)

	//
	// Test that locking fails properly when unable to acquire lock
	//

	// Reset variables used
	err = nil
	retCode = -512

	lf := flock.NewFlock(t.lockFile)
	c.Assert(lf, Not(IsNil))

	locked, err := lf.TryLock()
	c.Assert(err, IsNil)
	c.Assert(locked, Equals, true)

	retCode, _, _, err = handleCommand(t.h)
	c.Assert(err, Not(IsNil))
	c.Check(err.Error(), Equals, fmt.Sprintf("failed to obtain lock on '%v': locked by another process", t.lockFile))
	c.Check(retCode, Equals, 200)

	//
	// Test that locking succeeds with a timeout
	//

	// Reset variables used
	err = nil
	retCode = -512

	t.h.opts.WaitSeconds = 5
	t.h.cmd = exec.Command("/bin/echo", "something")

	go func() {
		time.Sleep(time.Second * 3)
		lf.Unlock()
	}()

	retCode, _, _, err = handleCommand(t.h)
	c.Assert(err, IsNil)
	c.Check(retCode, Equals, 0)

	// clear the statsd return channel
	_, ok = <-t.out
	c.Assert(ok, Equals, true)
	_, ok = <-t.out
	c.Assert(ok, Equals, true)

	//
	// Test that locking fails when exceeding the timeout
	//

	// Reset variables used
	err = nil
	retCode = -512

	t.h.opts.WaitSeconds = 1
	t.h.cmd = exec.Command("/bin/echo", "something")

	locked, err = lf.TryLock()
	c.Assert(err, IsNil)
	c.Assert(locked, Equals, true)

	go func() {
		time.Sleep(time.Second * 3)
		lf.Unlock()
	}()

	retCode, _, _, err = handleCommand(t.h)
	c.Assert(err, Not(IsNil))
	c.Check(err.Error(), Equals, "timeout exceeded (1s) waiting for the file lock")
	c.Check(retCode, Equals, 200)

	//
	// Test that warning Dogstatsd events are emitted if a
	// command is taking too long to run
	//

	// Reset variables used
	err = nil
	retCode = -512

	t.h.opts.Lock = false
	t.h.opts.WarnAfter = 2

	t.h.cmd = exec.Command("/bin/sleep", "3")

	retCode, r, runTime, err = handleCommand(t.h)
	c.Assert(err, IsNil)
	c.Assert(retCode, Equals, 0)
	c.Check(len(r), Equals, 0)

	// clear the statsd return channel
	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	c.Check(
		string(stat),
		Equals,
		fmt.Sprintf(`_e{56,65}:Cron testCmd still running after 2 seconds on brainbox01|UUID: %v\nrunning for 2 seconds|k:%v|s:cronner|t:warning|#source_type:cronner,cronner_label_name:testCmd`, t.h.uuid, t.h.uuid),
	)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)

	match = timeStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	statFloat, err = strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Check(statFloat, Equals, runTime)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)

	match = retStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	retFloat, err = strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Check(retFloat, Equals, float64(0))

	//
	// Test passthru to stdout/stderr
	//

	// Reset variables used
	r = nil
	err = nil
	runTime = 0
	match = nil
	retCode = -512

	t.h.cmd = exec.Command("/bin/bash", "testdata/echo.sh")
	t.h.opts.Passthru = true

	// Capture stdout/stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	oReader, oWriter, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	eReader, eWriter, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = oWriter
	os.Stderr = eWriter

	// copy the output in a separate goroutine (non-blocking)
	outC := make(chan string)
	errC := make(chan string)

	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, oReader)
		outC <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, eReader)
		errC <- buf.String()
	}()

	retCode, r, runTime, err = handleCommand(t.h)

	// restore stdout/stderr back to normal state
	oWriter.Close()
	eWriter.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	stdout := <-outC
	stderr := <-errC

	// Check the output
	c.Assert(err, IsNil)
	c.Check(retCode, Equals, 0)
	c.Assert(len(r), Equals, 0)
	c.Assert(stdout, Equals, "stdout\nstdout\nstdout\nstdout\n")
	c.Assert(stderr, Equals, "stderr\nstderr\nstderr\nstderr\n")

	_, ok = <-t.out
	c.Assert(ok, Equals, true)
}

func (t *TestSuite) Test_emitEvent(c *C) {
	title := "TE"
	body := "B"
	label := "urmom"
	alertType := "info"
	t.h.opts.EventGroup = "testing"

	emitEvent(title, body, label, alertType, t.h)

	event, ok := <-t.out
	c.Assert(ok, Equals, true)

	eventStub := fmt.Sprintf("_e{%d,%d}:%v|%v|k:%v|s:cronner|t:%v|#source_type:cronner,cronner_label_name:urmom,cronner_group:testing", len(title), len(body), title, body, t.h.uuid, alertType)
	eventStr := string(event)

	c.Check(eventStr, Equals, eventStub)

	//
	// Test truncation
	//

	// generate a body that will be truncated
	body = randString(4100)
	title = "TE2"
	label = "awwyiss"
	alertType = "success"
	t.h.opts.EventGroup = ""

	emitEvent(title, body, label, alertType, t.h)

	event, ok = <-t.out
	c.Assert(ok, Equals, true)

	// simulate truncation and addition of the truncation messsage
	truncatedBody := fmt.Sprintf("%v...\\n=== OUTPUT TRUNCATED ===\\n%v", body[0:MaxBody/2], body[len(body)-((MaxBody/2)+1):len(body)-1])

	eventStub = fmt.Sprintf("_e{%d,%d}:%v|%v|k:%v|s:cronner|t:%v|#source_type:cronner,cronner_label_name:awwyiss", len(title), len(truncatedBody), title, truncatedBody, t.h.uuid, alertType)
	eventStr = string(event)

	c.Check(eventStr, Equals, eventStub)
}

func (t *TestSuite) Test_writeOutput(c *C) {
	tmpDir, err := ioutil.TempDir("/tmp", "cronner_test")
	c.Assert(err, IsNil)

	defer os.RemoveAll(tmpDir)

	filename := path.Join(tmpDir, fmt.Sprintf("outfile-%v.out", randString(8)))
	out := []byte("this is a test!")

	ok := writeOutput(filename, out, false)
	c.Assert(ok, Equals, true)

	stat, err := os.Stat(filename)
	c.Assert(err, IsNil)
	c.Check(stat.Mode(), Equals, os.FileMode(0400))

	file, err := os.Open(filename)
	c.Assert(err, IsNil)

	contents, err := ioutil.ReadAll(file)
	c.Assert(err, IsNil)
	c.Check(string(out), Equals, string(contents))
}
