package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"

	"github.com/codeskyblue/go-uuid"
	"github.com/nightlyone/lockfile"
	. "gopkg.in/check.v1"
)

func (t *TestSuite) Test_runCommand(c *C) {
	//
	// Test a command that finishes in 0.3 seconds
	//
	uuidStr := uuid.New()
	host := "brainbox01"
	cmd := exec.Command("/usr/bin/time", "-p", "/bin/sleep", "0.3")

	// runButts(cmd *exec.Cmd, label string, save bool, gs *godspeed.Godspeed, lock bool, lockDir string)
	retCode, r, time, err := runCommand(cmd, t.gs, t.a, host, uuidStr)
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
	cmd = nil
	time = 0
	match = nil
	timely = false
	retCode = -512
	timeRegex = nil

	cmd = exec.Command("/usr/bin/time", "-p", "/bin/sleep", "1")

	retCode, r, time, err = runCommand(cmd, t.gs, t.a, host, uuidStr)
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
	cmd = nil
	time = 0
	match = nil
	retCode = -512

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("/bin/false")
	case "darwin":
		cmd = exec.Command("/usr/bin/false")
	}

	retCode, r, time, err = runCommand(cmd, t.gs, t.a, host, uuidStr)
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
	// Test that no output is given
	//

	// Reset variables used
	r = nil
	err = nil
	cmd = nil
	time = 0
	match = nil
	retCode = -300

	cmd = exec.Command("/bin/echo", "something")

	t.a.AllEvents = false
	t.a.Lock = true

	lf, err := lockfile.New(path.Join(t.a.LockDir, "cronner-testCmd.lock"))
	c.Assert(err, IsNil)

	retCode, r, _, err = runCommand(cmd, t.gs, t.a, host, uuidStr)
	c.Assert(err, IsNil)
	c.Assert(retCode, Equals, 0)
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

	retCode, _, _, err = runCommand(cmd, t.gs, t.a, host, uuidStr)
	c.Assert(err, Not(IsNil))
	c.Check(err.Error(), Equals, "Locked by other process")
	c.Check(retCode, Equals, 200)

	//
	// Test that warning Dogstatsd events are emitted if a
	// command is taking too long to run
	//

	// Reset variables used
	err = nil
	retCode = -512

	t.a.Lock = false
	t.a.WarnAfter = 2

	cmd = exec.Command("/bin/sleep", "3")

	retCode, r, time, err = runCommand(cmd, t.gs, t.a, host, uuidStr)
	c.Assert(err, IsNil)
	c.Assert(retCode, Equals, 0)
	c.Check(len(r), Equals, 0)

	// clear the statsd return channel
	stat, ok = <-t.out
	c.Assert(ok, Equals, true)
	c.Check(
		string(stat),
		Equals,
		fmt.Sprintf(`_e{56,65}:Cron testCmd still running after 2 seconds on brainbox01|UUID: %v\nrunning for 2 seconds|k:%v|s:cron|t:warning|#source_type:cron,label_name:testCmd`, uuidStr, uuidStr),
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
