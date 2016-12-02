// Copyright 2014-2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package godspeed_test

import (
	"fmt"
	"time"

	// this is *C comes from
	. "gopkg.in/check.v1"
)

func (t *TestSuite) TestEvent(c *C) {
	//
	// test that adding tags to both Godspeed and the Event send all tags
	//

	//
	// test whether length validation works
	//
	body := make([]byte, 8200)

	for i := range body {
		body[i] = 'a'
	}

	err := t.g.Event("some event", string(body), nil, nil)
	c.Check(err, Not(IsNil))

	//
	// test whether title validation works
	//
	err = t.g.Event("", "s", nil, nil)
	c.Assert(err, Not(IsNil))
	c.Check(err.Error(), Equals, "title must have at least one character")

	//
	// test whether body validation works
	//
	err = t.g.Event("s", "", nil, nil)
	c.Assert(err, Not(IsNil))
	c.Check(err.Error(), Equals, "body must have at least one character")

	//
	// general tests
	//
	err = t.g.Event("some\nother event", "some\nbody", nil, nil)
	c.Assert(err, IsNil)

	a, ok := <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "_e{17,10}:some\\nother event|some\\nbody")

	//
	// test that 'date_happened' value gets passed
	//
	unix := time.Now().Unix()

	m := make(map[string]string)
	m["date_happened"] = fmt.Sprintf("%d", unix)

	err = t.g.Event("a", "b", m, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, fmt.Sprintf("_e{1,1}:a|b|d:%d", unix))

	//
	// test that 'hostname' value gets passed
	//
	m = make(map[string]string)
	m["hostname"] = "tes|t01"

	err = t.g.Event("b", "c", m, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "_e{1,1}:b|c|h:test01")

	//
	// test that 'aggregation_key' value gets passed
	//
	m = make(map[string]string)
	m["aggregation_key"] = "xyz"

	err = t.g.Event("c", "d", m, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "_e{1,1}:c|d|k:xyz")

	//
	// test that 'priority' value gets passed
	//
	m = make(map[string]string)
	m["priority"] = "low"

	err = t.g.Event("d", "e", m, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "_e{1,1}:d|e|p:low")

	//
	// test that 'source_type_name' value gets passed
	//
	m = make(map[string]string)
	m["source_type_name"] = "cassandra"

	err = t.g.Event("e", "f", m, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "_e{1,1}:e|f|s:cassandra")

	//
	// test that 'alert_type' value gets passed
	//
	m = make(map[string]string)
	m["alert_type"] = "info"

	err = t.g.Event("f", "g", m, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "_e{1,1}:f|g|t:info")

	//
	// test that adding all values makes sure that all get passed
	//
	m = make(map[string]string)
	m["date_happened"] = fmt.Sprintf("%d", unix)
	m["hostname"] = "test01"
	m["aggregation_key"] = "xyz"
	m["priority"] = "low"
	m["source_type_name"] = "cassandra"
	m["alert_type"] = "info"

	err = t.g.Event("g", "h", m, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, fmt.Sprintf("_e{1,1}:g|h|d:%d|h:test01|k:xyz|p:low|s:cassandra|t:info", unix))

	//
	// test that adding tags only to the event works
	//
	err = t.g.Event("h", "i", nil, []string{"test8", "test9"})
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "_e{1,1}:h|i|#test8,test9")

	//
	// test that adding the tags to the Godspeed instance sends them with the event
	//
	t.g.AddTags([]string{"test0", "test1"})
	c.Assert(len(t.g.Tags), Equals, 2)

	err = t.g.Event("i", "j", nil, nil)
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "_e{1,1}:i|j|#test0,test1")

	//
	// test that adding tags to both Godspeed and the Event send all tags
	//
	err = t.g.Event("j", "k", nil, []string{"test8", "test9"})
	c.Assert(err, IsNil)

	a, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(a), Equals, "_e{1,1}:j|k|#test0,test1,test8,test9")
}
