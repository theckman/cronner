// Copyright 2014-2015 PagerDuty, Inc.
// All rights reserved - Do not redistribute!

// Package main is the main thing, man.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/PagerDuty/godspeed"
	"github.com/codeskyblue/go-uuid"
	"github.com/tideland/goas/v3/logger"
)

// MaxBody is the maximum length of a event body
const MaxBody = 4096

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

	// get the *exec.Cmd instance
	cmd := exec.Command(cmdParts[0], args...)

	uuidStr := uuid.New()

	if opts.AllEvents {
		// emit a DD event to indicate we are starting the job
		emitEvent(fmt.Sprintf("Cron %v starting on %v", opts.Label, hostname), fmt.Sprintf("UUID: %v\n", uuidStr), opts.Label, "info", uuidStr, gs)
	}

	// run the command and return the output as well as the return status
	ret, out, wallRtMs, err := runCommand(cmd, gs, opts)

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
	if alertType == "error" && opts.LogFail {
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
