// Copyright 2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"

	"github.com/tideland/golib/logger"

	. "gopkg.in/check.v1"
)

func (t *TestSuite) Test_binArgs_parse(c *C) {
	var output string
	var err error

	verOut := fmt.Sprintf("cronner v%s built with %s\nCopyright 2015 PagerDuty, Inc.; released under the BSD 3-Clause License\n", Version, runtime.Version())

	args := &binArgs{}
	cli := []string{}

	output, err = args.parse(cli)
	c.Assert(err, Not(IsNil))
	c.Check(len(output), Equals, 0)
	c.Check(err.Error(), Equals, "cron label '' is invalid, it can only be alphanumeric with underscores, periods, and spaces")

	args = &binArgs{}
	cli = []string{"-l", "test"}

	output, err = args.parse(cli)
	c.Assert(err, Not(IsNil))
	c.Check(len(output), Equals, 0)
	c.Check(err.Error(), Equals, "you must specify a command to run either using by adding it to the end, or using the command flag")

	args = &binArgs{}
	cli = []string{"-V"}

	output, err = args.parse(cli)
	c.Assert(err, IsNil)
	c.Check(output, Equals, verOut)
	c.Check(args.Version, Equals, true)

	args = &binArgs{}
	cli = []string{"--version"}

	output, err = args.parse(cli)
	c.Assert(err, IsNil)
	c.Check(output, Equals, verOut)
	c.Check(args.Version, Equals, true)

	args = &binArgs{}
	cli = []string{
		"-l", "test",
		"--", "/bin/true",
	}

	output, err = args.parse(cli)
	c.Assert(err, IsNil)
	c.Check(len(output), Equals, 0)
	c.Check(args.LockDir, Equals, "/var/lock")
	c.Check(args.AllEvents, Equals, false)
	c.Check(args.FailEvent, Equals, false)
	c.Check(args.LogFail, Equals, false)
	c.Check(args.EventGroup, Equals, "")
	c.Check(args.Lock, Equals, false)
	c.Check(args.LogPath, Equals, "/var/log/cronner")
	c.Check(args.LogLevel, Equals, "error")
	c.Check(args.Namespace, Equals, "cronner")
	c.Check(args.Sensitive, Equals, false)
	c.Check(args.Version, Equals, false)
	c.Check(args.WarnAfter, Equals, uint64(0))
	c.Check(args.WaitSeconds, Equals, uint64(0))

	args = &binArgs{}
	cli = []string{
		"-d", "/var/testlock",
		"-e",
		"-E",
		"-F",
		"-G", "test_group",
		"-k",
		"-l", "test",
		"-L", "info",
		"-N", "testcronner",
		"-s",
		"-w", "42",
		"-W", "84",
		"--", "/bin/true",
	}

	output, err = args.parse(cli)
	c.Assert(err, IsNil)

	// because we're parsing args we've just overridden this in the parser
	// so set it back to the value from SetUpSuite()
	logger.SetLevel(logger.LevelFatal)

	c.Check(len(output), Equals, 0)
	c.Check(args.LockDir, Equals, "/var/testlock")
	c.Check(args.AllEvents, Equals, true)
	c.Check(args.FailEvent, Equals, true)
	c.Check(args.LogFail, Equals, true)
	c.Check(args.EventGroup, Equals, "test_group")
	c.Check(args.Lock, Equals, true)
	c.Check(args.Label, Equals, "test")
	c.Check(args.LogLevel, Equals, "info")
	c.Check(args.Namespace, Equals, "testcronner")
	c.Check(args.Sensitive, Equals, true)
	c.Check(args.Version, Equals, false)
	c.Check(args.WarnAfter, Equals, uint64(42))
	c.Check(args.WaitSeconds, Equals, uint64(84))
	c.Assert(len(args.Args.Command), Equals, 1)
	c.Check(args.Args.Command[0], Equals, "/bin/true")

	args = &binArgs{}
	cli = []string{
		"--lock-dir", "/var/testlock",
		"--event",
		"--event-fail",
		"--log-fail",
		"--event-group", "test_group",
		"--lock",
		"--label", "test",
		"--log-path", "/var/log/testcronner",
		"--log-level", "info",
		"--namespace", "testcronner",
		"--sensitive",
		"--warn-after", "42",
		"--wait-secs", "84",
		"--", "/bin/true",
	}

	output, err = args.parse(cli)
	c.Assert(err, IsNil)
	logger.SetLevel(logger.LevelFatal)

	c.Check(len(output), Equals, 0)
	c.Check(args.LockDir, Equals, "/var/testlock")
	c.Check(args.AllEvents, Equals, true)
	c.Check(args.FailEvent, Equals, true)
	c.Check(args.LogFail, Equals, true)
	c.Check(args.EventGroup, Equals, "test_group")
	c.Check(args.Lock, Equals, true)
	c.Check(args.Label, Equals, "test")
	c.Check(args.LogPath, Equals, "/var/log/testcronner")
	c.Check(args.LogLevel, Equals, "info")
	c.Check(args.Namespace, Equals, "testcronner")
	c.Check(args.Sensitive, Equals, true)
	c.Check(args.Version, Equals, false)
	c.Check(args.WarnAfter, Equals, uint64(42))
	c.Check(args.WaitSeconds, Equals, uint64(84))
	c.Assert(len(args.Args.Command), Equals, 1)
	c.Check(args.Args.Command[0], Equals, "/bin/true")

	args = &binArgs{}
	cli = []string{
		"--lock-dir=/var/testlock",
		"--event-group=test_group",
		"--label=test",
		"--log-path=/var/log/testcronner",
		"--log-level=info",
		"--namespace=testcronner",
		"--warn-after=42",
		"--wait-secs=84",
		"--", "/bin/true",
	}

	output, err = args.parse(cli)
	c.Assert(err, IsNil)
	logger.SetLevel(logger.LevelFatal)

	c.Check(len(output), Equals, 0)
	c.Check(args.LockDir, Equals, "/var/testlock")
	c.Check(args.EventGroup, Equals, "test_group")
	c.Check(args.Label, Equals, "test")
	c.Check(args.LogPath, Equals, "/var/log/testcronner")
	c.Check(args.LogLevel, Equals, "info")
	c.Check(args.Namespace, Equals, "testcronner")
	c.Check(args.WarnAfter, Equals, uint64(42))
	c.Check(args.WaitSeconds, Equals, uint64(84))
	c.Assert(len(args.Args.Command), Equals, 1)
	c.Check(args.Args.Command[0], Equals, "/bin/true")
}
