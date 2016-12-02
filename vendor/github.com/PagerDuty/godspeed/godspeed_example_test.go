// Copyright 2014-2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package godspeed_test

import (
	"fmt"

	"github.com/PagerDuty/godspeed"
)

func ExampleNew() {
	g, err := godspeed.New(godspeed.DefaultHost, godspeed.DefaultPort, false)

	if err != nil {
		// handle error
	}

	defer g.Conn.Close()

	err = g.Gauge("example.stat", 1, nil)

	if err != nil {
		// handle error
	}
}

func ExampleNewDefault() {
	g, err := godspeed.NewDefault()

	if err != nil {
		// handle error
	}

	defer g.Conn.Close()

	g.Gauge("example.stat", 1, nil)
}

func ExampleGodspeed_AddTag() {
	// be sure to handle the error
	g, _ := godspeed.NewDefault()

	defer g.Conn.Close()

	g.AddTag("example1")

	tags := g.AddTag("example2")

	fmt.Println(tags)
	// Output: [example1 example2]
}

func ExampleGodspeed_AddTags() {
	g, _ := godspeed.NewDefault()

	defer g.Conn.Close()

	newTags := []string{"production", "example"}

	tags := g.AddTags(newTags)

	fmt.Println(tags)
	// Output: [production example]
}

func ExampleGodspeed_SetNamespace() {
	g, _ := godspeed.NewDefault()

	defer g.Conn.Close()

	namespace := "example"

	g.SetNamespace(namespace)

	fmt.Println(g.Namespace)
	// Output: example
}
