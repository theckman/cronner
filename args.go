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

// args is for argument parsing
type args struct {
	Label     string `short:"l" long:"label" default:"" description:"name for cron job to be used in statsd emissions and DogStatsd events. alphanumeric only; cronner will lowercase it"`
	Cmd       string `short:"c" long:"command" default:"" description:"(deprecated; use positional args) command to run (please use full path) and its args; executed as user running cronner"`
	AllEvents bool   `short:"e" long:"event" default:"false" description:"emit a start and end datadog event"`
	FailEvent bool   `short:"E" long:"event-fail" default:"false" description:"only emit an event on failure"`
	LogFail   bool   `short:"F" long:"log-fail" default:"false" description:"when a command fails, log its full output (stdout/stderr) to the log directory using the UUID as the filename"`
	LogPath   string `long:"log-path" default:"/var/log/cronner/" description:"where to place the log files for command output (path for -l/--log-on-fail output)"`
	LogLevel  string `short:"L" long:"log-level" default:"error" description:"set the level at which to log at [none|error|info|debug]"`
	Sensitive bool   `short:"s" long:"sensitive" default:"false" description:"specify whether command output may contain sensitive details, this only avoids it being printed to stderr"`
	Lock      bool   `short:"k" long:"lock" default:"false" description:"lock based on label so that multiple commands with the same label can not run concurrently"`
	LockDir   string `short:"d" long:"lock-dir" default:"/var/lock" description:"the directory where lock files will be placed"`
	Namespace string `short:"N" long:"namespace" default:"cronner" description:"namespace for statsd emissions, value is prepended to metric name by statsd client"`
	Args      struct {
		Command []string `positional-arg-name:"command [arguments]"`
	} `positional-args:"yes" required:"true"`
}

// parse function configures the go-flags parser and runs it
// it also does some light input validation
func (a *args) parse() error {
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
			fmt.Printf("%v", err.Error())
			os.Exit(0)
		}
	}

	r := regexp.MustCompile("^[a-zA-Z0-9_\\.]+$")

	if !r.MatchString(a.Label) {
		return fmt.Errorf("cron label '%v' is invalid, it can only be alphanumeric with underscores and periods", a.Label)
	}

	if a.Cmd == "" {
		if len(a.Args.Command) == 0 {
			return fmt.Errorf("you must specify a command to run either using by adding it to the end, or using the command flag")
		}
		a.Cmd = strings.Join(a.Args.Command, " ")
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
		return fmt.Errorf("%v is not a known log level, try none, debug, info, or error", a.LogLevel)
	}
	logger.SetLevel(logLevel)

	return nil
}
