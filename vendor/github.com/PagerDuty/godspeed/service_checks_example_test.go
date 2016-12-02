// Copyright 2014-2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package godspeed_test

import (
	"fmt"
	"os"

	"github.com/PagerDuty/godspeed"
)

func ExampleGodspeed_ServiceCheck() {
	// check the error
	g, _ := godspeed.NewDefault()

	defer g.Conn.Close()

	service := "Nagios Service"
	status := 0 // OK

	// if you don't want these, pass nil to the function instead
	optionals := make(map[string]string)
	optionals["service_check_message"] = "down"
	optionals["timestamp"] = "1431484263"

	// if you don't want these, pass nil to the function instead
	tags := []string{"some:tag"}

	err := g.ServiceCheck(service, status, optionals, tags)

	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
	}
}
