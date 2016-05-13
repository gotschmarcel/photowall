// +build osx

// Copyright 2016 Marcel Gotsch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func setSystemWallpaper() {
	wpFile := filepath.Join(cacheDir, wallpaperName)
	wpFile, err := filepath.Abs(wpFile)
	fatalIf(err)

	script := fmt.Sprintf("tell application %q to set desktop picture to POSIX file %q", "Finder", wpFile)

	cmd := exec.Command("/usr/bin/osascript", "-e", script)
	fatalIf(cmd.Run())
}
