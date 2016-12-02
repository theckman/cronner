// Copyright 2014-2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package godspeed_test

import "github.com/PagerDuty/godspeed"

func Example() {
	// this uses the default host and port (127.0.0.1:8125)
	g, err := godspeed.NewDefault()

	if err != nil {
		// handle error
	}

	// defer closing the connection
	defer g.Conn.Close()

	g.AddTags([]string{"example", "example2"})

	err = g.Incr("example.run_count", nil)

	// all emission method calls should return an error object
	// you should probably check it
	if err != nil {
		// handle error
	}

	// this returns an error object too, just omitting for brevity
	g.Gauge("example.gauge", 42, nil)
}
