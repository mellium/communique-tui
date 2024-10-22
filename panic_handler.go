// Copyright 2024 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"runtime/debug"
)

var uiShutdown = func() {}

// panicHandler should be deferred as the first thing in all goroutines started
// anywhere in the system.
// This will allow us to shut down and clean up the UI before printing the stack
// trace, thus averting a mess of text all over the screen.
// If we decide to save/send stack traces later this will also give us a central
// place to do that.
func panicHandler() {
	if r := recover(); r != nil {
		uiShutdown()
		fmt.Fprintf(os.Stderr, "%s\n", debug.Stack())
		fmt.Fprintln(os.Stderr, "----")
		panic(r)
	}
}
