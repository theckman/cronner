// Copyright 2014-2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package godspeed_test

import (
	"math/rand"

	"github.com/PagerDuty/godspeed"

	// this is *C comes from
	. "gopkg.in/check.v1"
)

var chars = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func randString(n uint) string {
	b := make([]rune, n)

	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}

	return string(b)
}

func (t *TestSuite) TestSend(c *C) {
	gs, err := godspeed.New("127.0.0.1", 8125, true)
	c.Assert(err, IsNil)

	//
	// Test whether auto-truncation works
	//

	// add a bunch of tags the pad the body with a lot of content
	for i := 0; i < 2100; i++ {
		gs.AddTag(randString(3))
	}

	err = gs.Send("test.metric", "c", 42, 1, nil)
	c.Assert(err, IsNil)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)
	c.Check(len(a), Equals, godspeed.MaxBytes)

	gs.Conn.Close()
	gs = nil

	//
	// test whether sending a plain metric works
	//
	err = t.g.Send("testing.metric", "ms", 256.512, 1, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "testing.metric:256.512|ms")

	//
	// test whether sending a large metric works
	//
	err = t.g.Send("testing.metric", "g", 5536650702696, 1, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "testing.metric:5536650702696|g")

	//
	// test whether sending a metric with a sample rate works
	//
	err = t.g.Send("testing.metric", "ms", 256.512, 0.99, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "testing.metric:256.512|ms|@0.99")

	//
	// test whether metrics are properly sent with the namespace
	//
	t.g.SetNamespace("godspeed")

	err = t.g.Send("testing.metric", "ms", 512.1024, 1, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "godspeed.testing.metric:512.1024|ms")

	//
	// test that adding a tag to the instance sends it with the metric
	//
	t.g.AddTag("test")

	err = t.g.Send("testing.metric", "ms", 512.1024, 1, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "godspeed.testing.metric:512.1024|ms|#test")

	//
	// test whether adding a second tag causes both to get sent with the stat
	//
	t.g.AddTag("test1")

	err = t.g.Send("testing.metric", "ms", 512.1024, 1, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "godspeed.testing.metric:512.1024|ms|#test,test1")

	//
	// test whether adding multiple tags sends all tags with the metric
	//
	t.g.AddTags([]string{"test2", "test3"})

	err = t.g.Send("testing.metric", "ms", 512.1024, 1, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "godspeed.testing.metric:512.1024|ms|#test,test1,test2,test3")

	//
	// test that adding metrics to the stat sends all instance tags with provided tags appended
	//
	err = t.g.Send("testing.metric", "ms", 512.1024, 1, []string{"test4", "test5", "test3"})
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "godspeed.testing.metric:512.1024|ms|#test,test1,test2,test3,test4,test5")

	//
	// test that adding tags to a single metric doesn't persist on future stats
	//
	err = t.g.Send("testing.metric", "ms", 512.1024, 1, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "godspeed.testing.metric:512.1024|ms|#test,test1,test2,test3")

	//
	// test that a failure is returned when autoTruncate is false, and the body is larger than MAX_BYTES
	//
	for i := 0; i < 2100; i++ {
		t.g.AddTag(randString(3))
	}

	err = t.g.Send("test.metric", "c", 42, 1, nil)
	c.Assert(err, Not(IsNil))
}

func (t *TestSuite) TestCount(c *C) {
	err := t.g.Count("test.count", 1, nil)
	c.Assert(err, IsNil)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "test.count:1|c")
}

func (t *TestSuite) TestIncr(c *C) {
	err := t.g.Incr("test.incr", nil)
	c.Assert(err, IsNil)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "test.incr:1|c")
}

func (t *TestSuite) TestDecr(c *C) {
	err := t.g.Decr("test.decr", nil)
	c.Assert(err, IsNil)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "test.decr:-1|c")
}

func (t *TestSuite) TestGauge(c *C) {
	err := t.g.Gauge("test.gauge", 42, nil)
	c.Assert(err, IsNil)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "test.gauge:42|g")
}

func (t *TestSuite) TestHistogram(c *C) {
	err := t.g.Histogram("test.hist", 84, nil)
	c.Assert(err, IsNil)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "test.hist:84|h")
}

func (t *TestSuite) TestTiming(c *C) {
	err := t.g.Timing("test.timing", 2054, nil)
	c.Assert(err, IsNil)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "test.timing:2054|ms")
}

func (t *TestSuite) TestSet(c *C) {
	err := t.g.Set("test.set", 10, nil)
	c.Assert(err, IsNil)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "test.set:10|s")
}

func (t *TestSuite) BenchmarkIncr(c *C) {
	t.g.SetNamespace("namespace")
	t.g.AddTags([]string{"a:1111", "b:2"})
	err := t.g.Incr("bench.incr", []string{"c:333", "d:444", "e:555555555"})
	c.Assert(err, IsNil)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "namespace.bench.incr:1|c|#a:1111,b:2,c:333,d:444,e:555555555")
	for i := 0; i < c.N; i++ {
		t.g.Incr("bench.incr", []string{"c:333", "d:444", "e:555555555"})
	}
}
