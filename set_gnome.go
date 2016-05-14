// +build lnx_gnome

// Copyright 2016 Marcel Gotsch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "os/exec"

func setSystemWallpaperCmd(file string) *exec.Cmd {
	return exec.Command("gconftool-2", "-t", "str", "-s", "/desktop/gnome/background/picture_filename", file)
}
