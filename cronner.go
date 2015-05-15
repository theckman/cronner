// Copyright 2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

// Package main is the main thing, man.
package main

import (
	"os"
	"os/exec"
	"strings"

	"github.com/PagerDuty/godspeed"
	"github.com/codeskyblue/go-uuid"
	"github.com/tideland/goas/v3/logger"
)

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
	err := opts.parse()

	// make sure parsing didn't bomb
	if err != nil {
		logger.Errorf("error: %v\n", err)
		os.Exit(1)
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

	// split the command in to its binary and arguments
	cmdParts := strings.Split(opts.Cmd, " ")

	// build the args slice
	var args []string
	if len(cmdParts) > 1 {
		args = cmdParts[1:]
	}

	handler := &cmdHandler{
		opts:     opts,
		hostname: hostname,
		gs:       gs,
		uuid:     uuid.New(),
		cmd:      exec.Command(cmdParts[0], args...),
	}

	ret, _, _, err := handleCommand(handler)

	if err != nil {
		logger.Errorf(err.Error())
	}

	os.Exit(ret)
}
