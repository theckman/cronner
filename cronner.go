// Copyright 2014 PagerDuty, Inc.
// All rights reserved - Do not redistribute!

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/PagerDuty/godspeed"
	"github.com/codeskyblue/go-uuid"
	"github.com/jessevdk/go-flags"
)

// MaxBody is the maximum length of a event body
const MaxBody = 4096

// args is for argument parsing
type args struct {
	Label     string `short:"l" long:"label" default:"" description:"name for cron job to be used in statsd emissions and DogStatsd events. alphanumeric only; cronner will lowercase it"`
	Cmd       string `short:"c" long:"command" default:"" description:"command to run (please use full path) and its args; executed as user running cronner"`
	AllEvents bool   `short:"e" long:"event" default:"false" description:"emit a start and end datadog event"`
	FailEvent bool   `short:"E" long:"event-fail" default:"false" description:"only emit an event on failure"`
}

// parse function configures the go-flags parser and runs it
// it also does some light input validation
func (a *args) parse() error {
	p := flags.NewParser(a, flags.HelpFlag|flags.PassDoubleDash)

	_, err := p.Parse()

	// determine if there was a parsing error
	// unfortunately, help message is returned as an error
	if err != nil {
		if !strings.Contains(err.Error(), "Usage") {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(-1)
		} else {
			fmt.Printf("%v\n", err.Error())
			os.Exit(0)
		}
	}

	r := regexp.MustCompile("^[a-zA-Z0-9_\\.]+$")

	if !r.MatchString(a.Label) {
		return fmt.Errorf("cron label '%v' is invalid, it can only be alphanumeric with underscores and periods", a.Label)
	}

	if a.Cmd == "" {
		return fmt.Errorf("you must specify a command to run")
	}

	// lowercase the metric to try and encourage sanity
	a.Label = strings.ToLower(a.Label)

	return nil
}

func runCommand(cmd *exec.Cmd, label string, gs *godspeed.Godspeed) (int, []byte, float64, error) {
	var b bytes.Buffer

	// comnbine stdout and stderr to the same buffer
	cmd.Stdout = &b
	cmd.Stderr = &b

	// log start time
	s := time.Now().UTC()

	err := cmd.Run()

	// This next section computes the wallclock run time in ms.
	// However, there is the unfortunate limitation in that
	// it uses the clock that gets adjusted by ntpd. Within pure
	// Go, I don't have access to CLOCK_MONOTONIC_RAW.
	//
	// However, based on our usage I don't think we care about it
	// being off by a few milliseconds.
	t := time.Since(s).Seconds() * 1000

	var ret int

	// if the command failed we want the exit status code
	// and to change the state variables to their failure values
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			status := ee.Sys().(syscall.WaitStatus)
			ret = status.ExitStatus()
		} else {
			ret = -1
		}
	}

	// emit the metric for how long it took us and return code
	gs.Timing(fmt.Sprintf("cron.%v.time", label), t, nil)
	gs.Gauge(fmt.Sprintf("cron.%v.exit_code", label), float64(ret), nil)

	return ret, b.Bytes(), t, err
}

// emit a godspeed (dogstatsd) event
func emitEvent(title, body, alertType, uuidStr string, g *godspeed.Godspeed) {
	var buf bytes.Buffer

	// if the event's body is bigger than MaxBody
	if len(body) > MaxBody {
		// push the first MaxBody/2 bytes in to the buffer
		buf.WriteString(body[0 : MaxBody/2])

		// add indication of truncated output to the buffer
		buf.WriteString("...\n=== OUTPUT TRUNCATED ===\n")

		// add the last 1024 bytes to the buffer
		buf.WriteString(body[len(body)-((MaxBody/2)+1) : len(body)-1])

		body = string(buf.Bytes())
	}

	fields := make(map[string]string)
	fields["source_type_name"] = "cron"

	if len(alertType) > 0 {
		fields["alert_type"] = alertType
	}

	if len(uuidStr) > 0 {
		fields["aggregation_key"] = uuidStr
	}

	g.Event(title, body, fields, nil)
}

func main() {
	// get and parse the command line options
	opts := &args{}
	err := opts.parse()

	// make sure parsing didn't bomb
	if err != nil {
		// print the error to stderr and exit -1
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(-1)
	}

	// build a Godspeed client
	gs, err := godspeed.NewDefault()

	// make sure nothing went wrong with Godspeed
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(-1)
	}

	gs.SetNamespace("pagerduty")

	// get the hostname and validate nothing happened
	hostname, err := os.Hostname()

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(-1)
	}

	// split the command in to its binary and arguments
	cmdParts := strings.Split(opts.Cmd, " ")

	// build the args slice
	var args []string
	if len(cmdParts) > 1 {
		args = cmdParts[1:]
	}

	// get the *exec.Cmd instance
	cmd := exec.Command(cmdParts[0], args...)

	var uuidStr string

	if opts.AllEvents {
		uuidStr = uuid.New()
		// emit a DD event to indicate we are starting the job
		emitEvent(fmt.Sprintf("Cron %v starting on %v", opts.Label, hostname), "job starting", "info", uuidStr, gs)
	}

	// run the command and return the output as well as the return status
	ret, out, wallRtMs, err := runCommand(cmd, opts.Label, gs)

	// default variables are for success
	// we change them later if there was a failure
	msg := "succeeded"
	alertType := "success"

	// if the command failed change the state variables to their failure values
	if err != nil {
		msg = "failed"
		alertType = "error"
	}

	if opts.AllEvents || (opts.FailEvent && alertType == "error") {
		// build the pieces of the completion event
		title := fmt.Sprintf("Cron %v %v in %.5f seconds on %v", opts.Label, msg, wallRtMs/1000, hostname)

		body := fmt.Sprintf("exit code: %d\n", ret)
		if err != nil {
			er := regexp.MustCompile("^exit status ([-]?\\d)")

			// do not show the 'more:' line, if the line is just telling us
			// what the exit code is
			if !er.MatchString(err.Error()) {
				body = fmt.Sprintf("%vmore: %v\n", body, err.Error())
			}
		}

		var cmdOutput string

		if len(out) > 0 {
			cmdOutput = string(out)
		} else {
			cmdOutput = "(none)"
		}

		body = fmt.Sprintf("%voutput: %v", body, cmdOutput)

		if uuidStr == "" {
			uuidStr = uuid.New()
		}

		emitEvent(title, body, alertType, uuidStr, gs)
	}
}
