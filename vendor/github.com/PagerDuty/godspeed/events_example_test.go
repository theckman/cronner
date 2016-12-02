// Copyright 2014-2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package godspeed_test

import (
	"fmt"

	"github.com/PagerDuty/godspeed"
)

func ExampleGodspeed_Event() {
	// make sure to handle the error
	g, _ := godspeed.NewDefault()

	defer g.Conn.Close()

	title := "Nginx service restart"
	text := "The Nginx service has been restarted"

	// the optionals are for the optional arguments available for an event
	// http://docs.datadoghq.com/guides/dogstatsd/#fields
	optionals := make(map[string]string)
	optionals["alert_type"] = "info"
	optionals["source_type_name"] = "nginx"

	addlTags := []string{"source_type:nginx"}

	err := g.Event(title, text, optionals, addlTags)

	if err != nil {
		fmt.Println("err:", err)
	}
}
