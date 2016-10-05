// Copyright 2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

// Package main is the main thing, man.
package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/PagerDuty/godspeed"
	"github.com/codeskyblue/go-uuid"
	"github.com/tideland/golib/logger"
)

// Version is the program's version string
const Version = "0.2.6"

type cmdHandler struct {
	gs       *godspeed.Godspeed
	opts     *binArgs
	cmd      *exec.Cmd
	uuid     string
	hostname string
}

func main() {
	logger.SetLogger(logger.NewStandardLogger(os.Stderr))

	// get and parse the command line options
	opts := &binArgs{}
	output, err := opts.parse(nil)

	// make sure parsing didn't bomb
	if err != nil {
		logger.Errorf("error: %v\n", err)
		os.Exit(1)
	}

	// if parsing had output, print it and exit 0
	if len(output) > 0 {
		fmt.Print(output)
		os.Exit(0)
	}

	// build a Godspeed client
	gs, err := godspeed.NewDefault()

	// make sure nothing went wrong with Godspeed
	if err != nil {
		logger.Errorf("error: %v\n", err)
		os.Exit(1)
	}

	gs.SetNamespace(opts.Namespace)

	// get the hostname and validate nothing happened
	hostname, err := os.Hostname()

	if err != nil {
		logger.Errorf("error: %v\n", err)
		os.Exit(1)
	}

	handler := &cmdHandler{
		opts:     opts,
		hostname: hostname,
		gs:       gs,
		uuid:     uuid.New(),
		cmd:      exec.Command(opts.Cmd, opts.CmdArgs...),
	}

	ret, _, _, err := handleCommand(handler)

	if err != nil {
		logger.Errorf(err.Error())
	}

	os.Exit(ret)
}
