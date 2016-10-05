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

	const Arg0 = "/usr/loca/bin/cronner"

	//
	// assert that label is required and is validated
	//
	args := &binArgs{}
	cli := []string{Arg0}

	output, err = args.parse(cli)
	c.Assert(err, Not(IsNil))
	c.Check(len(output), Equals, 0)
	c.Check(err.Error(), Equals, "cron label '' is invalid, it can only be alphanumeric with underscores, periods, and spaces")

	args = &binArgs{}
	cli = []string{
		Arg0,
		"-l", "invalid^label",
	}

	output, err = args.parse(cli)
	c.Assert(err, Not(IsNil))
	c.Check(len(output), Equals, 0)
	c.Check(err.Error(), Equals, "cron label 'invalid^label' is invalid, it can only be alphanumeric with underscores, periods, and spaces")

	//
	// assert that a command is required
	//
	args = &binArgs{}
	cli = []string{
		Arg0,
		"-l", "test",
	}

	output, err = args.parse(cli)
	c.Assert(err, Not(IsNil))
	c.Check(len(output), Equals, 0)
	c.Check(err.Error(), Equals, "you must specify a command to run either using by adding it to the end, or using the command flag")

	//
	// assert that version (-v/--version) printing works
	//
	args = &binArgs{}
	cli = []string{
		Arg0,
		"-V",
	}

	verOut := fmt.Sprintf("cronner v%s built with %s\nCopyright 2015 PagerDuty, Inc.; released under the BSD 3-Clause License\n", Version, runtime.Version())

	output, err = args.parse(cli)
	c.Assert(err, IsNil)
	c.Check(output, Equals, verOut)
	c.Check(args.Version, Equals, true)

	args = &binArgs{}
	cli = []string{
		Arg0,
		"--version",
	}

	output, err = args.parse(cli)
	c.Assert(err, IsNil)
	c.Check(output, Equals, verOut)
	c.Check(args.Version, Equals, true)

	//
	// assert the default values
	//
	args = &binArgs{}
	cli = []string{
		Arg0,
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
	c.Check(args.Passthru, Equals, false)
	c.Check(args.Sensitive, Equals, false)
	c.Check(args.Version, Equals, false)
	c.Check(args.WarnAfter, Equals, uint64(0))
	c.Check(args.WaitSeconds, Equals, uint64(0))

	//
	// assert that the short flags work
	//
	args = &binArgs{}
	cli = []string{
		Arg0,
		"-d", "/var/testlock",
		"-e",
		"-E",
		"-F",
		"-G", "test_group",
		"-k",
		"-l", "test",
		"-L", "info",
		"-N", "testcronner",
		"-p",
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
	c.Check(args.Passthru, Equals, true)
	c.Check(args.Sensitive, Equals, true)
	c.Check(args.Version, Equals, false)
	c.Check(args.WarnAfter, Equals, uint64(42))
	c.Check(args.WaitSeconds, Equals, uint64(84))
	c.Check(args.Cmd, Equals, "/bin/true")
	c.Check(len(args.CmdArgs), Equals, 0)

	//
	// assert that long flags work
	//
	args = &binArgs{}
	cli = []string{
		Arg0,
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
		"--passthru",
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
	c.Check(args.Passthru, Equals, true)
	c.Check(args.Sensitive, Equals, true)
	c.Check(args.Version, Equals, false)
	c.Check(args.WarnAfter, Equals, uint64(42))
	c.Check(args.WaitSeconds, Equals, uint64(84))
	c.Check(args.Cmd, Equals, "/bin/true")
	c.Check(len(args.CmdArgs), Equals, 0)

	//
	// assert that long flags work with --flag=value syntax
	//
	args = &binArgs{}
	cli = []string{
		Arg0,
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
	c.Check(args.Cmd, Equals, "/bin/true")
	c.Check(len(args.CmdArgs), Equals, 0)

	//
	// argument parsing regression tests
	//

	//
	// parse() function should always discard element 0 in the slice.
	//
	args = &binArgs{}
	cli = []string{
		"--lock-dir=/var/testlock",
		"--event-group=test_group",
		"--label=test",
		"--", "/bin/true",
	}

	output, err = args.parse(cli)
	c.Assert(err, IsNil)
	logger.SetLevel(logger.LevelFatal)

	c.Check(len(output), Equals, 0)
	c.Check(args.LockDir, Not(Equals), "/var/testlock")
	c.Check(args.EventGroup, Equals, "test_group")
	c.Check(args.Label, Equals, "test")
	c.Check(args.Cmd, Equals, "/bin/true")
	c.Check(len(args.CmdArgs), Equals, 0)

	//
	// parse() function should allow spaces in command line arguments
	//
	args = &binArgs{}
	cli = []string{
		Arg0,
		"--label=test",
		"--", "/bin/true", `some string`,
	}

	output, err = args.parse(cli)
	c.Assert(err, IsNil)
	logger.SetLevel(logger.LevelFatal)

	c.Check(len(output), Equals, 0)
	c.Check(args.Label, Equals, "test")
	c.Check(args.Cmd, Equals, "/bin/true")
	c.Assert(len(args.CmdArgs), Equals, 1)
	c.Check(args.CmdArgs[0], Equals, "some string")
}
