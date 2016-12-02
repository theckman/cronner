// Copyright 2014-2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package godspeed_test

import (
	"fmt"

	"github.com/PagerDuty/godspeed"
	. "gopkg.in/check.v1"
)

func (t *TestSuite) TestServiceCheck(c *C) {
	//
	// Test that a proper datagram is formed with
	// with no fields or tags included
	//
	err := t.g.ServiceCheck("testSvc", 0, nil, nil)
	c.Assert(err, IsNil)

	dgram, ok := <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(dgram), Equals, "_sc|testSvc|0")

	//
	// Test that the datagram is valid with tags
	//
	err = t.g.ServiceCheck("testSvc", 1, nil, []string{"tag:test", "tag2:testing"})
	c.Assert(err, IsNil)

	dgram, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(dgram), Equals, "_sc|testSvc|1|#tag:test,tag2:testing")

	//
	// Test that the datagram is valid with the
	// service_check_message field
	//
	fields := make(map[string]string)
	fields["service_check_message"] = "server on fire"

	err = t.g.ServiceCheck("testSvc", 1, fields, nil)
	c.Assert(err, IsNil)

	dgram, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(dgram), Equals, "_sc|testSvc|1|m:server on fire")

	//
	// Test that the datagram is valid with the timestamp field
	//
	fields["timestamp"] = "1431484263"

	err = t.g.ServiceCheck("testSvc", 1, fields, nil)
	c.Assert(err, IsNil)

	dgram, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(dgram), Equals, "_sc|testSvc|1|m:server on fire|d:1431484263")

	//
	// Test that the datagram is valid with the hostname field
	//
	fields["hostname"] = "brainbox01"

	err = t.g.ServiceCheck("testSvc", 2, fields, nil)
	c.Assert(err, IsNil)

	dgram, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(dgram), Equals, "_sc|testSvc|2|m:server on fire|d:1431484263|h:brainbox01")

	//
	// Test that the datagram is valid when we put it all together
	//
	err = t.g.ServiceCheck("testSvc", 3, fields, []string{"tag:test", "tag2:testing"})
	c.Assert(err, IsNil)

	dgram, ok = <-t.o
	c.Assert(ok, Equals, true)
	c.Check(string(dgram), Equals, "_sc|testSvc|3|m:server on fire|d:1431484263|h:brainbox01|#tag:test,tag2:testing")

	//
	// Test that invalid service names trigger an error
	//
	err = t.g.ServiceCheck("", 0, nil, nil)
	c.Assert(err, Not(IsNil))
	c.Check(err.Error(), Equals, "service name must have at least one character")

	err = t.g.ServiceCheck("some|pipe", 0, nil, nil)
	c.Assert(err, Not(IsNil))
	c.Check(err.Error(), Equals, "service name 'some|pipe' may not include pipe character ('|')")

	//
	// Test that invalid service statuses trigger an error
	//
	err = t.g.ServiceCheck("testSvc", 4, nil, nil)
	c.Assert(err, Not(IsNil))
	c.Check(err.Error(), Equals, "unknown service status (4); known values: 0,1,2,3")

	err = t.g.ServiceCheck("testSvc", -1, nil, nil)
	c.Assert(err, Not(IsNil))
	c.Check(err.Error(), Equals, "unknown service status (-1); known values: 0,1,2,3")

	//
	// Test that making the datagram huge causes a failure
	//
	svcTitle := randString(godspeed.MaxBytes)

	err = t.g.ServiceCheck(svcTitle, 0, nil, nil)
	c.Assert(err, Not(IsNil))
	c.Check(err.Error(), Equals, fmt.Sprintf("error sending %s service check, packet larger than 8192 (8198)", svcTitle))
}
