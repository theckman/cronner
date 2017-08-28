// Copyright 2016-2017 Tim Heckman
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.
//
// +build go1.9

//
// NOTE(theckman):
// This file is used to define a constant indicating that go1.9+ is being used.
//
// cronner requires the monotonic time source introduced in Go 1.9. See:
//
// - https://github.com/golang/go/issues/12914
//

package main

// This constant is a hack to communicate that this software must be built
// against Go 1.9+.
//
// runner.go references this constant, and the build tags of this file will
// make it appear undefined on anything older than go1.9.
const cronnerRequiresAtleastGoVersion19 = uint8(0)
