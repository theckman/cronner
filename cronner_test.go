// Copyright 2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"path"
	"testing"
	"time"

	"github.com/PagerDuty/godspeed"
	"github.com/codeskyblue/go-uuid"
	"github.com/tideland/golib/logger"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct {
	l        *net.UDPConn
	ctrl     chan int
	out      chan []byte
	lockFile string
	h        *cmdHandler
}

var _ = Suite(&TestSuite{})

func (t *TestSuite) SetUpSuite(c *C) {
	// suppress application logging
	logger.SetLevel(logger.LevelFatal)

	workingDir := c.MkDir()

	t.h = &cmdHandler{
		hostname: "brainbox01",
		uuid:     uuid.New(),
		opts: &binArgs{
			Label:   "testCmd",
			LogFail: true,
			LogPath: workingDir,
			LockDir: workingDir,
		},
	}

	var err error

	t.h.gs, err = godspeed.NewDefault()
	c.Assert(err, IsNil)
	t.h.gs.SetNamespace("cronner")

	t.lockFile = path.Join(t.h.opts.LockDir, "cronner-testCmd.lock")
}

func (t *TestSuite) TearDownSuite(c *C) {
	t.h.gs.Conn.Close()
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

	time.Sleep(time.Millisecond * 10)
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
