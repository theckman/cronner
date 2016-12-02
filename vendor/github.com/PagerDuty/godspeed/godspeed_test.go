// Copyright 2014-2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package godspeed_test

import (
	"net"
	"testing"
	"time"

	"github.com/PagerDuty/godspeed"
	"github.com/PagerDuty/godspeed/gspdtest"

	// this is *C comes from
	. "gopkg.in/check.v1"
)

const closedChan = "return channel (out) closed prematurely"

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct {
	g *godspeed.Godspeed
	l *net.UDPConn
	c chan int
	o chan []byte
}

var _ = Suite(&TestSuite{})

func (t *TestSuite) SetUpTest(c *C) {
	gs, err := godspeed.NewDefault()
	c.Assert(err, IsNil)
	t.g = gs

	t.l, t.c, t.o = gspdtest.BuildListener(8125)
	go gspdtest.Listener(t.l, t.c, t.o)
}

func (t *TestSuite) TearDownTest(c *C) {
	t.l.Close()
	close(t.c)
	t.g.Conn.Close()
	time.Sleep(time.Millisecond * 10)
}

func testBasicFunc(t *TestSuite, c *C, g *godspeed.Godspeed) {
	err := g.Send("test.metric", "c", 1, 1, nil)
	c.Assert(err, IsNil)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)

	b := []byte("test.metric:1|c")
	c.Check(string(a), Equals, string(b))
	c.Check(len(g.Tags), Equals, 0)
	c.Check(g.Namespace, Equals, "")
}

func (t *TestSuite) TestNew(c *C) {
	// build Godspeed
	var g *godspeed.Godspeed
	g, err := godspeed.New("127.0.0.1", 8125, false)
	c.Assert(err, IsNil)

	defer g.Conn.Close()

	// test defined basic functionality
	testBasicFunc(t, c, g)
}

func (t *TestSuite) TestNewDefault(c *C) {
	var g *godspeed.Godspeed
	g, err := godspeed.NewDefault()
	c.Assert(err, IsNil)

	defer g.Conn.Close()

	testBasicFunc(t, c, g)
}

func (t *TestSuite) TestAddTag(c *C) {
	c.Assert(len(t.g.Tags), Equals, 0)

	t.g.AddTag("test")
	c.Assert(len(t.g.Tags), Equals, 1)
	c.Check(t.g.Tags[0], Equals, "test")

	t.g.AddTag("test2")
	t.g.AddTag("test") // verify tags are de-duped

	c.Assert(len(t.g.Tags), Equals, 2)
	c.Check(t.g.Tags[0], Equals, "test")
	c.Check(t.g.Tags[1], Equals, "test2")
}

func (t *TestSuite) TestAddTags(c *C) {
	c.Assert(len(t.g.Tags), Equals, 0)

	tags := []string{"test1", "test2", "test1"}

	t.g.AddTags(tags)
	c.Assert(len(t.g.Tags), Equals, 2)

	// match content
	for i := range t.g.Tags {
		c.Check(t.g.Tags[i], Equals, tags[i])
	}

	tags2 := []string{"test3", "test4", "test5", "test4"}
	tags = append(tags, tags2...)

	t.g.AddTags(tags2)
	c.Assert(len(t.g.Tags), Equals, 5)

	control := []string{"test1", "test2", "test3", "test4", "test5"}

	// match content
	for i := range t.g.Tags {
		c.Check(t.g.Tags[i], Equals, control[i])
	}
}

func (t *TestSuite) TestSetNamespace(c *C) {
	c.Check(t.g.Namespace, Equals, "")
	t.g.SetNamespace("heckman")
	c.Check(t.g.Namespace, Equals, "heckman")
}
