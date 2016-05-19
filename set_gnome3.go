// +build lnx_gnome3

// Copyright 2016 Marcel Gotsch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os/exec"
)

var setLockScreen bool

func init() {
	flag.BoolVar(&setLockScreen, "set-lockscreen", false, "Set lock screen image")
}

func systemUpdate(file string) error {
	if setWallpaper {
		err := exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", "file://"+file).Run()
		if err != nil {
			return fmt.Errorf("Unable to set wallpaper, %s", err.Error())
		}
	}

	if setLockScreen {
		err := exec.Command("gsettings", "set", "org.gnome.desktop.screensaver", "picture-uri", "file://"+file).Run()
		if err != nil {
			return fmt.Errorf("Unable to set lock screen, %s", err.Error())
		}
	}

	return nil
}
