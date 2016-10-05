// Copyright 2015 PagerDuty, Inc, et al. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/tideland/golib/logger"
)

// binArgs is for argument parsing
type binArgs struct {
	Cmd         string   // this is not a command line flag, but rather parsed results
	CmdArgs     []string // this is not a command line flag, also parsed results
	LockDir     string   `short:"d" long:"lock-dir" default:"/var/lock" description:"the directory where lock files will be placed"`
	AllEvents   bool     `short:"e" long:"event" description:"emit a start and end datadog event"`
	FailEvent   bool     `short:"E" long:"event-fail" description:"only emit an event on failure"`
	LogFail     bool     `short:"F" long:"log-fail" description:"when a command fails, log its full output (stdout/stderr) to the log directory using the UUID as the filename"`
	EventGroup  string   `short:"G" long:"event-group" value-name:"<group>" description:"emit a cronner_group:<group> tag with Datadog events, does not get sent with statsd metrics"`
	Lock        bool     `short:"k" long:"lock" description:"lock based on label so that multiple commands with the same label can not run concurrently"`
	Label       string   `short:"l" long:"label" description:"name for cron job to be used in statsd emissions and DogStatsd events. alphanumeric only; cronner will lowercase it"`
	LogPath     string   `long:"log-path" default:"/var/log/cronner" description:"where to place the log files for command output (path for -F/--log-fail output)"`
	LogLevel    string   `short:"L" long:"log-level" default:"error" description:"set the level at which to log at [none|error|info|debug]"`
	Namespace   string   `short:"N" long:"namespace" default:"cronner" description:"namespace for statsd emissions, value is prepended to metric name by statsd client"`
	Passthru    bool     `short:"p" long:"passthru" description:"passthru stdout/stderr to controlling tty"`
	Sensitive   bool     `short:"s" long:"sensitive" description:"specify whether command output may contain sensitive details, this only avoids it being printed to stderr"`
	Version     bool     `short:"V" long:"version" description:"print the version string and exit"`
	WarnAfter   uint64   `short:"w" long:"warn-after" default:"0" value-name:"N" description:"emit a warning event every N seconds if the job hasn't finished, set to 0 to disable"`
	WaitSeconds uint64   `short:"W" long:"wait-secs" default:"0" description:"how long to wait for the file lock for"`
	Args        struct {
		Command []string `positional-arg-name:"-- command [arguments]"`
	} `positional-args:"yes" required:"true"`
}

var argsLabelRegex = regexp.MustCompile(`^[a-zA-Z0-9_\. ]+$`)

// parse function configures the go-flags parser and runs it
// it also does some light input validation
//
// the args parameter is meant to be the entirety of os.Args
func (a *binArgs) parse(args []string) (string, error) {
	if args == nil {
		args = os.Args
	}

	p := flags.NewParser(a, flags.HelpFlag|flags.PassDoubleDash)

	_, err := p.ParseArgs(args[1:])

	// determine if there was a parsing error
	// unfortunately, help message is returned as an error
	if err != nil {
		// determine whether this was a help message by doing a type
		// assertion of err to *flags.Error and check the error type
		// if it was a help message, do not return an error
		if errType, ok := err.(*flags.Error); ok {
			if errType.Type == flags.ErrHelp {
				return err.Error(), nil
			}
		}

		return "", err
	}

	if a.Version {
		out := fmt.Sprintf("cronner v%s built with %s\nCopyright 2015 PagerDuty, Inc.; released under the BSD 3-Clause License\n", Version, runtime.Version())
		return out, nil
	}

	if !argsLabelRegex.MatchString(a.Label) {
		return "", fmt.Errorf("cron label '%v' is invalid, it can only be alphanumeric with underscores, periods, and spaces", a.Label)
	}

	if len(a.Args.Command) == 0 {
		return "", fmt.Errorf("you must specify a command to run either using by adding it to the end, or using the command flag")
	}
	a.Cmd = a.Args.Command[0]

	if len(a.Args.Command) > 1 {
		a.CmdArgs = a.Args.Command[1:]
	}

	// lowercase the metric and replace spaces with underscores
	// to try and encourage sanity
	a.Label = strings.Replace(strings.ToLower(a.Label), " ", "_", -1)

	var logLevel logger.LogLevel

	switch strings.ToLower(a.LogLevel) {
	case "none":
		logLevel = logger.LevelFatal
	case "error":
		logLevel = logger.LevelError
	case "info":
		logLevel = logger.LevelInfo
	case "debug":
		logLevel = logger.LevelDebug
	default:
		return "", fmt.Errorf("%v is not a known log level, try none, debug, info, or error", a.LogLevel)
	}
	logger.SetLevel(logLevel)

	return "", nil
}
