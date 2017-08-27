// Copyright 2015 PagerDuty, Inc., et al.
// Copyright 2016-2017 Tim Heckman
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.
//
// +build !go1.9

//
// NOTE(theckman):
// THIS DUMMY FILE IS FOR IF THE RUNTIME IS OLDER THAN 1.9
//
// CRONNER REQUIRES THE MONOTONIC TIME SOURCE INTRODUCED IN GO 1.9
//

package main

// handleCommand should never be invoked from this file due to the
// isThisBuiltAgainstAtLeastGo19 being undefined at build time.
//
// that said, trigger a panic just in case we do somehow manage to get here
func handleCommand(hndlr *cmdHandler) (int, []byte, float64, error) {
	panic("cronner requires it be built against go1.9+ to use the monotonic time source")
}
