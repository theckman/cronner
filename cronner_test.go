// Copyright 2014-2015 PagerDuty, Inc.
// All rights reserved - Do not redistribute!

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"testing"

	"github.com/PagerDuty/godspeed"
	"github.com/codeskyblue/go-uuid"
	"github.com/nightlyone/lockfile"
	"github.com/tideland/goas/v3/logger"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct {
	gs   *godspeed.Godspeed
	l    *net.UDPConn
	ctrl chan int
	out  chan []byte
}

var _ = Suite(&TestSuite{})

func (t *TestSuite) SetUpSuite(c *C) {
	var err error
	t.gs, err = godspeed.NewDefault()
	c.Assert(err, IsNil)
	t.gs.SetNamespace("pagerduty")
}

func (t *TestSuite) TearDownSuite(c *C) {
	t.gs.Conn.Close()
}

func (t *TestSuite) SetUpTest(c *C) {
	t.l, t.ctrl, t.out = buildListener(8125)

	// this goroutine will get cleaned up by the
	// TearDownTest function
	go listener(t.l, t.ctrl, t.out)
}

func (t *TestSuite) TearDownTest(c *C) {
	close(t.ctrl)
	t.l.Close()
}

func (t *TestSuite) Test_runCommand(c *C) {
	//
	// Test a command that finishes in 0.3 seconds
	//
	cmd := exec.Command("/usr/bin/time", "-p", "/bin/sleep", "0.3")

	retCode, r, time, err := runCommand(cmd, "testCmd", true, t.gs, false, "")
	c.Assert(err, IsNil)
	c.Assert(retCode, Equals, 0)

	stat, ok := <-t.out
	c.Assert(ok, Equals, true)

	timeStatRegex := regexp.MustCompile("^pagerduty.cron.testCmd.time:([0-9\\.]+)\\|ms$")
	match := timeStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	statFloat, err := strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Assert(statFloat, Equals, time)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)

	retStatRegex := regexp.MustCompile("^pagerduty.cron.testCmd.exit_code:([0-9\\.]+)\\|g$")
	match = retStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	retFloat, err := strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Assert(retFloat, Equals, float64(0))

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
	c.Assert(match[0][2], Equals, "0.30")

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
	retCode = 0
	timeRegex = nil

	cmd = exec.Command("/usr/bin/time", "-p", "/bin/sleep", "1")

	retCode, r, time, err = runCommand(cmd, "testCmd", true, t.gs, false, "")
	c.Assert(err, IsNil)
	c.Assert(retCode, Equals, 0)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)

	match = timeStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	statFloat, err = strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Assert(statFloat, Equals, time)

	if time > 1000 && time < 1020 {
		timely = true
	}
	c.Assert(timely, Equals, true)

	timeRegex = regexp.MustCompile("((?m)^real[[:space:]]+([0-9\\.]+)$)")
	match = timeRegex.FindAllStringSubmatch(string(r), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 3)
	c.Assert(match[0][2], Equals, "1.00")

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)

	match = retStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	retFloat, err = strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Assert(retFloat, Equals, float64(0))

	//
	// Test a valid return code is given
	//

	// Reset variables used
	r = nil
	err = nil
	cmd = nil
	time = 0
	match = nil
	retCode = 0

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("/bin/false")
	case "darwin":
		cmd = exec.Command("/usr/bin/false")
	}

	retCode, r, time, err = runCommand(cmd, "testCmd", true, t.gs, false, "")
	c.Assert(err, Not(IsNil))
	c.Assert(retCode, Equals, 1)

	_, ok = <-t.out
	c.Assert(ok, Equals, true)

	stat, ok = <-t.out
	c.Assert(ok, Equals, true)

	match = retStatRegex.FindAllStringSubmatch(string(stat), -1)
	c.Assert(len(match), Equals, 1)
	c.Assert(len(match[0]), Equals, 2)

	retFloat, err = strconv.ParseFloat(match[0][1], 64)
	c.Assert(err, IsNil)
	c.Assert(retFloat, Equals, float64(1))

	//
	// Test that no output is given
	//

	// Reset variables used
	r = nil
	err = nil
	cmd = nil
	time = 0
	match = nil
	retCode = 0

	cmd = exec.Command("/bin/echo", "something")

	retCode, r, _, err = runCommand(cmd, "testCmd", false, t.gs, false, "")
	c.Assert(err, IsNil)
	c.Assert(retCode, Equals, 0)
	c.Check(len(r), Equals, 0)

	// clear the statsd return channel
	_, ok = <-t.out
	c.Assert(ok, Equals, true)
	_, ok = <-t.out
	c.Assert(ok, Equals, true)
}

func (t *TestSuite) Test_withLock_doesLock(c *C) {
	// suppress errors when locking
	logger.SetLevel(logger.LevelFatal)

	label := "be.ok"
	lockDir := "/tmp"

	lockPath := path.Join(lockDir, fmt.Sprintf("cronner-%v.lock", label))

	lf, err := lockfile.New(lockPath)
	c.Assert(err, IsNil)

	err = lf.TryLock()
	c.Assert(err, IsNil)

	cmd := exec.Command("/bin/true")
	ret, _, _ := withLock(cmd, label, t.gs, true, lockDir)

	lf.Unlock()

	c.Assert(ret, Equals, 200)
}

func (t *TestSuite) Test_emitEvent(c *C) {
	title := "TE"
	body := "B"
	label := "urmom"
	alertType := "info"
	uuidStr := uuid.New()

	emitEvent(title, body, label, alertType, uuidStr, t.gs)

	event, ok := <-t.out
	c.Assert(ok, Equals, true)

	eventStub := fmt.Sprintf("_e{%d,%d}:%v|%v|k:%v|s:cron|t:%v|#source_type:cron,label_name:urmom", len(title), len(body), title, body, uuidStr, alertType)
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

	emitEvent(title, body, label, alertType, uuidStr, t.gs)

	event, ok = <-t.out
	c.Assert(ok, Equals, true)

	// simulate truncation and addition of the truncation messsage
	truncatedBody := fmt.Sprintf("%v...\\n=== OUTPUT TRUNCATED ===\\n%v", body[0:MaxBody/2], body[len(body)-((MaxBody/2)+1):len(body)-1])

	eventStub = fmt.Sprintf("_e{%d,%d}:%v|%v|k:%v|s:cron|t:%v|#source_type:cron,label_name:awwyiss", len(title), len(truncatedBody), title, truncatedBody, uuidStr, alertType)
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

//
// Cronner testing helper functions
//
var chars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

func randString(size int) string {
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = chars[rand.Intn(len(chars))]
	}
	return string(buf)
}

func listener(l *net.UDPConn, ctrl <-chan int, c chan<- []byte) {
	for {
		select {
		case _, ok := <-ctrl:
			if !ok {
				close(c)
				return
			}
		default:
			buffer := make([]byte, 8193)

			_, err := l.Read(buffer)

			if err != nil {
				continue
			}

			c <- bytes.Trim(buffer, "\x00")
		}
	}
}

func buildListener(port uint16) (*net.UDPConn, chan int, chan []byte) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", port))

	if err != nil {
		panic(fmt.Sprintf("getting address for test listener failed, bailing out. Here's everything I know: %v", err))
	}

	l, err := net.ListenUDP("udp", addr)

	if err != nil {
		panic(fmt.Sprintf("unable to listen for traffic: %v", err))
	}

	return l, make(chan int), make(chan []byte)
}
