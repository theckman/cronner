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
	"path"
	"testing"

	"github.com/PagerDuty/godspeed"
	"github.com/codeskyblue/go-uuid"
	"github.com/tideland/goas/v3/logger"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct {
	gs   *godspeed.Godspeed
	l    *net.UDPConn
	ctrl chan int
	out  chan []byte
	a    *binArgs
}

var _ = Suite(&TestSuite{})

func (t *TestSuite) SetUpSuite(c *C) {
	// suppress application logging
	logger.SetLevel(logger.LevelFatal)

	var err error

	t.gs, err = godspeed.NewDefault()
	c.Assert(err, IsNil)
	t.gs.SetNamespace("cronner")

	t.a = &binArgs{
		Label:     "testCmd",
		AllEvents: true,
		LockDir:   c.MkDir(),
	}
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
