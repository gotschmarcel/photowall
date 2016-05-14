// +build osx

// Copyright 2016 Marcel Gotsch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os/exec"
)

func setSystemWallpaperCmd(file string) *exec.Cmd {
	script := fmt.Sprintf("tell application %q to set desktop picture to POSIX file %q", "Finder", file)
	return exec.Command("/usr/bin/osascript", "-e", script)
}
