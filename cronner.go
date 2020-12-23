// Copyright 2015 PagerDuty, Inc, et al.
// Copyright 2016-2017 Tim Heckman
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

// Package main is the main thing, man.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PagerDuty/godspeed"
	"github.com/codeskyblue/go-uuid"
	"github.com/tideland/golib/logger"
)

// Version is the program's version string
const Version = "1.0.0"

type cmdHandler struct {
	gs               *godspeed.Godspeed
	opts             *binArgs
	cmd              *exec.Cmd
	uuid             string
	hostname         string
	parentEventTags  []string
	parentMetricTags []string
}

var cronnerEventEnvVars = []string{
	"CRONNER_PARENT_UUID",
	"CRONNER_PARENT_EVENT_GROUP",
}

var cronnerMetricEnvVars = []string{
	"CRONNER_PARENT_GROUP",
	"CRONNER_PARENT_NAMESPACE",
	"CRONNER_PARENT_LABEL",
}

func parseEnv(vars []string) []string {
	if len(vars) == 0 {
		return nil
	}

	tags := make([]string, len(vars))

	var count int

	for i, key := range vars {
		if val := os.Getenv(key); len(val) > 0 {
			tags[i] = fmt.Sprintf("%s:%s", strings.ToLower(key), val)
			count++
		}
	}

	if count == 0 {
		return nil
	}

	return tags[0:count]
}

func parseEnvForParent() (eventTags, metricTags []string) {
	// get CRONNER_PARENT_UUID to see if we're a parent process
	if os.Getenv(cronnerEventEnvVars[0]) == "" {
		return
	}

	eventTags = parseEnv(cronnerEventEnvVars)
	metricTags = parseEnv(cronnerMetricEnvVars)

	return
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
	var gs *godspeed.Godspeed

	var StatsdHost string
	StatsdHost = godspeed.DefaultHost
	if opts.StatsdHost != "" {
		StatsdHost = opts.StatsdHost
	}

	var StatsdPort int
	StatsdPort = godspeed.DefaultPort
	if opts.StatsdPort != 0 {
		StatsdPort = opts.StatsdPort
	}

	gs, err = godspeed.New(StatsdHost, StatsdPort, false)

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

	handler.parentEventTags, handler.parentMetricTags = parseEnvForParent()

	ret, _, _, err := handleCommand(handler)

	if err != nil {
		logger.Errorf(err.Error())
	}

	os.Exit(ret)
}
