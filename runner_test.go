package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"

	"github.com/nightlyone/lockfile"
	. "gopkg.in/check.v1"
)

func (t *TestSuite) Test_runCommand(c *C) {
	//
	// Test a command that finishes in 0.3 seconds
	//
	t.h.cmd = exec.Command("/usr/bin/time", "-p", "/bin/sleep", "0.3")

	retCode, r, time, err := handleCommand(t.h)
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
	c.Check(statFloat, Equals, time)

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
	if time > 300 && time < 320 {
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
	time = 0
	match = nil
	timely = false
	retCode = -512
	timeRegex = nil

	t.h.cmd = exec.Command("/usr/bin/time", "-p", "/bin/sleep", "1")

	retCode, r, time, err = handleCommand(t.h)
	c.Assert(err, IsNil)
	c.Check(retCode, Equals, 0)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)

	match = timeStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	statFloat, err = strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Check(statFloat, Equals, time)

	if time > 1000 && time < 1020 {
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
	time = 0
	match = nil
	retCode = -512

	switch runtime.GOOS {
	case "linux":
		t.h.cmd = exec.Command("/bin/false")
	case "darwin":
		t.h.cmd = exec.Command("/usr/bin/false")
	}

	retCode, r, time, err = handleCommand(t.h)
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
	time = 0
	match = nil
	retCode = -512

	t.h.cmd = exec.Command("/bin/echo", "somevalue")
	t.h.opts.AllEvents = true

	retCode, r, time, err = handleCommand(t.h)
	c.Assert(err, IsNil)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	c.Check(
		string(stat),
		Equals,
		fmt.Sprintf(`_e{35,44}:Cron testCmd starting on brainbox01|UUID: %v\n|k:%v|s:cron|t:info|#source_type:cron,label_name:testCmd`, t.h.uuid, t.h.uuid),
	)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	match = timeStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)
	c.Check(strconv.FormatFloat(time, 'f', -1, 64), Equals, match[0][1])

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	c.Check(string(stat), Equals, "cronner.testCmd.exit_code:0|g")

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	c.Check(
		string(stat),
		Equals,
		fmt.Sprintf(`_e{55,77}:Cron testCmd succeeded in %.5f seconds on brainbox01|UUID: %v\nexit code: 0\noutput: somevalue\n|k:%v|s:cron|t:success|#source_type:cron,label_name:testCmd`, time/1000, t.h.uuid, t.h.uuid),
	)

	//
	// Test that no output is given
	//

	// Reset variables used
	r = nil
	err = nil
	time = 0
	match = nil

	t.h.cmd = exec.Command("/bin/echo", "something")

	t.h.opts.LogFail = false
	t.h.opts.Lock = true
	t.h.opts.AllEvents = false

	lf, err := lockfile.New(t.lockFile)
	c.Assert(err, IsNil)

	retCode, r, _, err = handleCommand(t.h)
	c.Assert(err, IsNil)
	c.Check(retCode, Equals, 0)
	c.Check(len(r), Equals, 0)

	// assert that the lockfile was removed
	_, err = os.Stat(string(lf))
	c.Assert(os.IsNotExist(err), Equals, true)

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

	err = lf.TryLock()
	c.Assert(err, IsNil)
	defer lf.Unlock()

	retCode, _, _, err = handleCommand(t.h)
	c.Assert(err, Not(IsNil))
	c.Check(err.Error(), Equals, fmt.Sprintf("failed to obtain lock on '%v': Locked by other process", t.lockFile))
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

	retCode, r, time, err = handleCommand(t.h)
	c.Assert(err, IsNil)
	c.Assert(retCode, Equals, 0)
	c.Check(len(r), Equals, 0)

	// clear the statsd return channel
	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	c.Check(
		string(stat),
		Equals,
		fmt.Sprintf(`_e{56,65}:Cron testCmd still running after 2 seconds on brainbox01|UUID: %v\nrunning for 2 seconds|k:%v|s:cron|t:warning|#source_type:cron,label_name:testCmd`, t.h.uuid, t.h.uuid),
	)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)

	match = timeStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	statFloat, err = strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Check(statFloat, Equals, time)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)

	match = retStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	retFloat, err = strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Check(retFloat, Equals, float64(0))
}

func (t *TestSuite) Test_emitEvent(c *C) {
	title := "TE"
	body := "B"
	label := "urmom"
	alertType := "info"

	emitEvent(title, body, label, alertType, t.h.uuid, t.h.gs)

	event, ok := <-t.out
	c.Assert(ok, Equals, true)

	eventStub := fmt.Sprintf("_e{%d,%d}:%v|%v|k:%v|s:cron|t:%v|#source_type:cron,label_name:urmom", len(title), len(body), title, body, t.h.uuid, alertType)
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

	emitEvent(title, body, label, alertType, t.h.uuid, t.h.gs)

	event, ok = <-t.out
	c.Assert(ok, Equals, true)

	// simulate truncation and addition of the truncation messsage
	truncatedBody := fmt.Sprintf("%v...\\n=== OUTPUT TRUNCATED ===\\n%v", body[0:MaxBody/2], body[len(body)-((MaxBody/2)+1):len(body)-1])

	eventStub = fmt.Sprintf("_e{%d,%d}:%v|%v|k:%v|s:cron|t:%v|#source_type:cron,label_name:awwyiss", len(title), len(truncatedBody), title, truncatedBody, t.h.uuid, alertType)
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
