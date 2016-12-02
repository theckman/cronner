// Copyright 2014-2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package godspeed_test

import "github.com/PagerDuty/godspeed"

func ExampleNewAsync() {
	a, err := godspeed.NewAsync(godspeed.DefaultHost, godspeed.DefaultPort, false)

	if err != nil {
		// handle error
	}

	defer a.Godspeed.Conn.Close()

	// add to the WaitGroup to make sure we are able to wait below
	a.W.Add(1)

	go a.Gauge("example.gauge", 1, nil, a.W)

	a.W.Wait()
}

func ExampleNewDefaultAsync() {
	a, err := godspeed.NewDefaultAsync()

	if err != nil {
		// handle error
	}

	defer a.Godspeed.Conn.Close()

	a.W.Add(1)

	go a.Gauge("example.gauge", 1, nil, a.W)

	a.W.Wait()
}

func ExampleAsyncGodspeed_Event() {
	a, _ := godspeed.NewDefaultAsync()

	defer a.Godspeed.Conn.Close()

	a.W.Add(1)

	go a.Event("example event", "something happened", nil, nil, a.W)

	a.W.Wait()
}

func ExampleAsyncGodspeed_Send() {
	a, _ := godspeed.NewDefaultAsync()

	defer a.Godspeed.Conn.Close()

	a.W.Add(1)

	go a.Send("example.stat", "g", 1, 1, nil, a.W)

	a.W.Wait()
}

func ExampleAsyncGodspeed_Count() {
	a, _ := godspeed.NewDefaultAsync()

	defer a.Godspeed.Conn.Close()

	a.W.Add(1)

	go a.Count("example.count", 42, nil, a.W)

	a.W.Wait()
}

func ExampleAsyncGodspeed_Gauge() {
	a, _ := godspeed.NewDefaultAsync()

	defer a.Godspeed.Conn.Close()

	a.W.Add(1)

	go a.Gauge("example.gauge", 1, nil, a.W)

	a.W.Wait()
}
