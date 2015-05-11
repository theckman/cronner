// Copyright 2014-2015 PagerDuty, Inc.
// All rights reserved - Do not redistribute!

package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/tideland/goas/v3/logger"
)

// binArgs is for argument parsing
type binArgs struct {
	Cmd        string // this is not a command line flag, but rather parsed results
	Label      string `short:"l" long:"label" default:"" description:"name for cron job to be used in statsd emissions and DogStatsd events. alphanumeric only; cronner will lowercase it"`
	Namespace  string `short:"N" long:"namespace" default:"cronner" description:"namespace for statsd emissions, value is prepended to metric name by statsd client"`
	EventGroup string `short:"G" long:"event-group" value-name:"<group>" description:"emit a cronner_group:<group> tag with Datadog events, does not get sent with statsd metrics"`
	AllEvents  bool   `short:"e" long:"event" default:"false" description:"emit a start and end datadog event"`
	FailEvent  bool   `short:"E" long:"event-fail" default:"false" description:"only emit an event on failure"`
	WarnAfter  uint64 `short:"w" long:"warn-after" default:"0" value-name:"N" description:"emit a warning event every N seconds if the job hasn't finished, set to 0 to disable"`
	LogFail    bool   `short:"F" long:"log-fail" default:"false" description:"when a command fails, log its full output (stdout/stderr) to the log directory using the UUID as the filename"`
	LogPath    string `long:"log-path" default:"/var/log/cronner/" description:"where to place the log files for command output (path for -l/--log-on-fail output)"`
	Sensitive  bool   `short:"s" long:"sensitive" default:"false" description:"specify whether command output may contain sensitive details, this only avoids it being printed to stderr"`
	Lock       bool   `short:"k" long:"lock" default:"false" description:"lock based on label so that multiple commands with the same label can not run concurrently"`
	LockDir    string `short:"d" long:"lock-dir" default:"/var/lock" description:"the directory where lock files will be placed"`
	LogLevel   string `short:"L" long:"log-level" default:"error" description:"set the level at which to log at [none|error|info|debug]"`
	Args       struct {
		Command []string `positional-arg-name:"command [arguments]"`
	} `positional-args:"yes" required:"true"`
}

// parse function configures the go-flags parser and runs it
// it also does some light input validation
func (a *binArgs) parse() error {
	p := flags.NewParser(a, flags.HelpFlag|flags.PassDoubleDash)
	//p.Usage = Usage

	_, err := p.Parse()

	// determine if there was a parsing error
	// unfortunately, help message is returned as an error
	if err != nil {
		if !strings.Contains(err.Error(), "Usage") {
			logger.Errorf("error: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Printf("%v", err)
			os.Exit(0)
		}
	}

	r := regexp.MustCompile("^[a-zA-Z0-9_\\.]+$")

	if !r.MatchString(a.Label) {
		return fmt.Errorf("cron label '%v' is invalid, it can only be alphanumeric with underscores and periods", a.Label)
	}

	if len(a.Args.Command) == 0 {
		return fmt.Errorf("you must specify a command to run either using by adding it to the end, or using the command flag")
	}
	a.Cmd = strings.Join(a.Args.Command, " ")

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
		return fmt.Errorf("%v is not a known log level, try none, debug, info, or error", a.LogLevel)
	}
	logger.SetLevel(logLevel)

	return nil
}
