// Copyright 2014 PagerDuty, Inc.
// All rights reserved - Do not redistribute!

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/PagerDuty/godspeed"
	"github.com/codeskyblue/go-uuid"
	"github.com/jessevdk/go-flags"
	"github.com/nightlyone/lockfile"
	"github.com/tideland/goas/v3/logger"
)

// MaxBody is the maximum length of a event body
const MaxBody = 4096
const intErrCode = 200

// args is for argument parsing
type args struct {
	Label     string `short:"l" long:"label" default:"" description:"name for cron job to be used in statsd emissions and DogStatsd events. alphanumeric only; cronner will lowercase it"`
	Cmd       string `short:"c" long:"command" default:"" description:"command to run (please use full path) and its args; executed as user running cronner"`
	AllEvents bool   `short:"e" long:"event" default:"false" description:"emit a start and end datadog event"`
	FailEvent bool   `short:"E" long:"event-fail" default:"false" description:"only emit an event on failure"`
	LogOnFail bool   `short:"F" long:"log-on-fail" default:"false" description:"when a command fails, log its full output (stdout/stderr) to the log directory using the UUID as the filename"`
	LogPath   string `long:"log-path" default:"/var/log/cronner/" description:"where to place the log files for command output (path for -l/--log-on-fail output)"`
	LogLevel  string `short:"L" long:"log-level" default:"error" description:"set the level at which to log at [none|error|info|debug]"`
	Sensitive bool   `short:"s" long:"sensitive" default:"false" description:"specify whether command output may contain sensitive details, this only avoids it being printed to stderr"`
	Lock      bool   `short:"k" long:"lock" default:"false" description:"lock based on label so that multiple commands with the same label can not run concurrently"`
	LockDir   string `short:"d" long:"lock-dir" default:"/var/lock" description:"the directory where lock files will be placed"`
}

// parse function configures the go-flags parser and runs it
// it also does some light input validation
func (a *args) parse() error {
	p := flags.NewParser(a, flags.HelpFlag|flags.PassDoubleDash)

	leftOvers, err := p.Parse()

	// determine if there was a parsing error
	// unfortunately, help message is returned as an error
	if err != nil {
		if !strings.Contains(err.Error(), "Usage") {
			logger.Errorf("error: %v\n", err)
			os.Exit(1)
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
		if len(leftOvers) == 0 {
			return fmt.Errorf("you must specify a command to run either using by adding it to the end, or using the command flag")
		}
		a.Cmd = strings.Join(leftOvers, " ")
	}

	// lowercase the metric to try and encourage sanity
	a.Label = strings.ToLower(a.Label)

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

func withLock(cmd *exec.Cmd, label string, gs *godspeed.Godspeed, lock bool, lockDir string) (int, float64, error) {
	var lf lockfile.Lockfile
	if lock {
		lockPath := path.Join(lockDir, fmt.Sprintf("cronner-%v.lock", label))

		lf, err := lockfile.New(lockPath)
		if err != nil {
			logger.Criticalf("Cannot init lock. reason: %v", err)
			return intErrCode, 0, err
		}

		err = lf.TryLock()
		if err != nil {
			logger.Criticalf("Cannot lock. reason: %v", err)
			return intErrCode, 0, err
		}
	}

	// log start time
	s := time.Now().UTC()

	cmdErr := cmd.Run()

	// This next section computes the wallclock run time in ms.
	// However, there is the unfortunate limitation in that
	// it uses the clock that gets adjusted by ntpd. Within pure
	// Go, I don't have access to CLOCK_MONOTONIC_RAW.
	//
	// However, based on our usage I don't think we care about it
	// being off by a few milliseconds.
	t := time.Since(s).Seconds() * 1000

	if lock {
		if err := lf.Unlock(); err != nil {
			logger.Criticalf("Cannot unlock. reason: %v", err)
			return intErrCode, t, err
		}
	}

	var ret int

	if cmdErr != nil {
		if ee, ok := cmdErr.(*exec.ExitError); ok {
			status := ee.Sys().(syscall.WaitStatus)
			ret = status.ExitStatus()
		} else {
			ret = intErrCode
		}
	}

	return ret, t, cmdErr
}

func runCommand(cmd *exec.Cmd, label string, save bool, gs *godspeed.Godspeed, lock bool, lockDir string) (int, []byte, float64, error) {
	var b bytes.Buffer

	if save {
		// comnbine stdout and stderr to the same buffer
		cmd.Stdout = &b
		cmd.Stderr = &b
	} else {
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	ret, t, err := withLock(cmd, label, gs, lock, lockDir)

	// emit the metric for how long it took us and return code
	gs.Timing(fmt.Sprintf("cron.%v.time", label), t, nil)
	gs.Gauge(fmt.Sprintf("cron.%v.exit_code", label), float64(ret), nil)

	return ret, b.Bytes(), t, err
}

// emit a godspeed (dogstatsd) event
func emitEvent(title, body, label, alertType, uuidStr string, g *godspeed.Godspeed) {
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

	tags := []string{"source_type:cron", fmt.Sprintf("label_name:%v", label)}

	g.Event(title, body, fields, tags)
}

func main() {
	logger.SetLogger(logger.NewStandardLogger(os.Stderr))

	// get and parse the command line options
	opts := &args{}
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

	gs.SetNamespace("pagerduty")

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

	// get the *exec.Cmd instance
	cmd := exec.Command(cmdParts[0], args...)

	uuidStr := uuid.New()

	if opts.AllEvents {
		// emit a DD event to indicate we are starting the job
		emitEvent(fmt.Sprintf("Cron %v starting on %v", opts.Label, hostname), fmt.Sprintf("UUID:%v\n", uuidStr), opts.Label, "info", uuidStr, gs)
	}

	var saveOutput bool

	if opts.AllEvents || opts.FailEvent || opts.LogOnFail {
		saveOutput = true
	}

	// run the command and return the output as well as the return status
	ret, out, wallRtMs, err := runCommand(cmd, opts.Label, saveOutput, gs, opts.Lock, opts.LockDir)

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

		body := fmt.Sprintf("UUID: %v\nexit code: %d\n", uuidStr, ret)
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

		emitEvent(title, body, opts.Label, alertType, uuidStr, gs)
	}

	// this code block is meant to be ran last
	if alertType == "error" && opts.LogOnFail {
		filename := path.Join(opts.LogPath, fmt.Sprintf("%v-%v.out", opts.Label, uuidStr))
		if !writeOutput(filename, out, opts.Sensitive) {
			os.Exit(1)
		}
	}
}

// bailOut is for failures during logfile writing
func bailOut(out []byte, sensitive bool) bool {
	if !sensitive {
		fmt.Fprintf(os.Stderr, "here is the output in hopes you are looking here:\n\n%v", string(out))
		os.Exit(1)
	}
	return false
}

// writeOutput saves the output (out) to the file specified
func writeOutput(filename string, out []byte, sensitive bool) bool {
	// check to see whehter or not the output file already exists
	// this should really never happen, but just in case it does...
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "flagrant error: output file '%v' already exists\n", filename)
		return bailOut(out, sensitive)
	}

	outFile, err := os.Create(filename)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening file to save command output: %v\n", err.Error())
		return bailOut(out, sensitive)
	}

	defer outFile.Close()

	if err = outFile.Chmod(0400); err != nil {
		fmt.Fprintf(os.Stderr, "error setting permissions (0400) on file '%v': %v\n", filename, err.Error())
		return bailOut(out, sensitive)
	}

	nwrt, err := outFile.Write(out)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing to file '%v': %v\n", filename, err.Error())
		return bailOut(out, sensitive)
	}

	if nwrt != len(out) {
		fmt.Fprintf(os.Stderr, "error writing to file '%v': number of bytes written not equal to output (total: %d, written: %d)\n", filename, len(out), nwrt)
		return bailOut(out, sensitive)
	}

	return true
}
