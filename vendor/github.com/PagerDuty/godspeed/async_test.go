// Copyright 2014-2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package godspeed_test

import (
	"fmt"
	"net"
	"time"

	"github.com/PagerDuty/godspeed"
	"github.com/PagerDuty/godspeed/gspdtest"

	// this is *C comes from
	. "gopkg.in/check.v1"
)

var extraTestTags = []string{"test8", "test9"}

type ATestSuite struct {
	g *godspeed.AsyncGodspeed
	l *net.UDPConn
	c chan int
	o chan []byte
}

var _ = Suite(&ATestSuite{})

func (t *ATestSuite) SetUpTest(c *C) {
	gs, err := godspeed.NewDefaultAsync()
	c.Assert(err, IsNil)
	t.g = gs

	t.g.SetNamespace("godspeed")
	t.g.AddTags([]string{"test0", "test1"})

	t.l, t.c, t.o = gspdtest.BuildListener(8125)
	go gspdtest.Listener(t.l, t.c, t.o)
}

func (t *ATestSuite) TearDownTest(c *C) {
	t.l.Close()
	close(t.c)
	t.g.Godspeed.Conn.Close()
	time.Sleep(time.Millisecond * 10)
}

func testAsyncBasicFunc(t *ATestSuite, c *C, g *godspeed.AsyncGodspeed) {
	g.AddTag("test0")
	g.SetNamespace("godspeed")

	g.W.Add(1)
	go g.Send("test.metric", "c", 1, 1, []string{"test1", "test2"}, g.W)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)

	b := []byte("godspeed.test.metric:1|c|#test0,test1,test2")
	c.Check(string(a), Equals, string(b))
	c.Check(len(g.Godspeed.Tags), Equals, 1)
	c.Check(g.Godspeed.Namespace, Equals, "godspeed")
}

func (t *ATestSuite) TestNewAsync(c *C) {
	var g *godspeed.AsyncGodspeed
	g, err := godspeed.NewAsync("127.0.0.1", 8125, false)
	c.Assert(err, IsNil)

	defer g.Godspeed.Conn.Close()

	testAsyncBasicFunc(t, c, g)
}

func (t *ATestSuite) TestNewDefaultAsync(c *C) {
	var g *godspeed.AsyncGodspeed
	g, err := godspeed.NewDefaultAsync()
	c.Assert(err, IsNil)

	defer g.Godspeed.Conn.Close()

	testAsyncBasicFunc(t, c, g)
}

func (t *ATestSuite) TestAsyncAddTags(c *C) {
	var g *godspeed.AsyncGodspeed
	g, err := godspeed.NewDefaultAsync()
	c.Assert(err, IsNil)

	c.Assert(len(g.Godspeed.Tags), Equals, 0)

	g.AddTag("testing0")
	c.Assert(len(g.Godspeed.Tags), Equals, 1)
	c.Check(g.Godspeed.Tags[0], Equals, "testing0")

	g.AddTags([]string{"testing1", "testing2", "testing0"})
	c.Assert(len(g.Godspeed.Tags), Equals, 3)
	c.Check(g.Godspeed.Tags[0], Equals, "testing0")
	c.Check(g.Godspeed.Tags[1], Equals, "testing1")
	c.Check(g.Godspeed.Tags[2], Equals, "testing2")
}

func (t *ATestSuite) TestSetNamespace(c *C) {
	var g *godspeed.AsyncGodspeed
	g, err := godspeed.NewDefaultAsync()
	c.Assert(err, IsNil)
	c.Check(g.Godspeed.Namespace, Equals, "")
	g.SetNamespace("heckman")
	c.Check(g.Godspeed.Namespace, Equals, "heckman")
}

func (t *ATestSuite) TestAsyncEvent(c *C) {
	t.g.AddTags([]string{"test0", "test1"})

	unix := time.Now().Unix()

	m := make(map[string]string)
	m["date_happened"] = fmt.Sprintf("%d", unix)
	m["hostname"] = "test01"
	m["aggregation_key"] = "xyz"
	m["priority"] = "low"
	m["source_type_name"] = "cassandra"
	m["alert_type"] = "info"

	t.g.W.Add(1)
	go t.g.Event("a", "b", m, []string{"test8", "test9"}, t.g.W)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)

	b := []byte(fmt.Sprintf("_e{1,1}:a|b|d:%d|h:test01|k:xyz|p:low|s:cassandra|t:info|#test0,test1,test8,test9", unix))
	c.Check(string(a), Equals, string(b))
}

func (t *ATestSuite) TestAsyncSend(c *C) {
	t.g.W.Add(1)
	go t.g.Send("test.stat", "g", 42, 0.99, extraTestTags, t.g.W)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)

	b := []byte("godspeed.test.stat:42|g|@0.99|#test0,test1,test8,test9")
	c.Check(string(a), Equals, string(b))
}

func (t *ATestSuite) TestAsyncServiceCheck(c *C) {
	fields := make(map[string]string)
	fields["service_check_message"] = "server on fire"
	fields["timestamp"] = "1431484263"
	fields["hostname"] = "brainbox01"

	t.g.W.Add(1)
	go t.g.ServiceCheck("testSvc", 0, fields, []string{"tag:test", "tag2:testing"}, t.g.W)

	dgram, ok := <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(dgram), Equals, "_sc|testSvc|0|m:server on fire|d:1431484263|h:brainbox01|#test0,test1,tag:test,tag2:testing")
}

func (t *ATestSuite) TestAsyncCount(c *C) {
	t.g.W.Add(1)
	go t.g.Count("test.count", 1, extraTestTags, t.g.W)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)

	b := []byte("godspeed.test.count:1|c|#test0,test1,test8,test9")
	c.Check(string(a), Equals, string(b))
}

func (t *ATestSuite) TestAsyncIncr(c *C) {
	t.g.W.Add(1)
	go t.g.Incr("test.incr", extraTestTags, t.g.W)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)

	b := []byte("godspeed.test.incr:1|c|#test0,test1,test8,test9")
	c.Check(string(a), Equals, string(b))
}

func (t *ATestSuite) TestAsyncDecr(c *C) {
	t.g.W.Add(1)
	go t.g.Decr("test.decr", extraTestTags, t.g.W)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)

	b := []byte("godspeed.test.decr:-1|c|#test0,test1,test8,test9")
	c.Check(string(a), Equals, string(b))
}

func (t *ATestSuite) TestAsyncGauge(c *C) {
	t.g.W.Add(1)
	go t.g.Gauge("test.gauge", 42, extraTestTags, t.g.W)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)

	b := []byte("godspeed.test.gauge:42|g|#test0,test1,test8,test9")
	c.Check(string(a), Equals, string(b))
}

func (t *ATestSuite) TestAsyncHistogram(c *C) {
	t.g.W.Add(1)
	go t.g.Histogram("test.hist", 2, extraTestTags, t.g.W)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)

	b := []byte("godspeed.test.hist:2|h|#test0,test1,test8,test9")
	c.Check(string(a), Equals, string(b))
}

func (t *ATestSuite) TestAsyncTiming(c *C) {
	t.g.W.Add(1)
	go t.g.Timing("test.timing", 3, extraTestTags, t.g.W)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)

	b := []byte("godspeed.test.timing:3|ms|#test0,test1,test8,test9")
	c.Check(string(a), Equals, string(b))
}

func (t *ATestSuite) TestAsyncSet(c *C) {
	t.g.W.Add(1)
	go t.g.Set("test.set", 4, extraTestTags, t.g.W)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)

	b := []byte("godspeed.test.set:4|s|#test0,test1,test8,test9")
	c.Check(string(a), Equals, string(b))
}
