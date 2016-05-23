# PhotoStream

Generate grid wallpapers based on photos from *Instagram* or *Tumblr*.

## Build & Install

Install *PhotoStream* with:

```bash
$ go get github.com/gotschmarcel/photostream
```

## Usage

The only required option is `-profile`. It specifies the account name from which the photos should be taken. The default
API is *Instagram*.

```bash
$ photostream -profile "jondoe"
```

*PhotoStream* provides a lot more options, such as background color, output size, spacing and more. To
get an overview of all available options run `$ photostream -h`

## Cron and System Wallpaper

Use *cron* to automatically update the wallpaper in regular intervals.

### Mac OS X

Here's a simple bash script which you can call from your cron job to regularly update the system
wallpaper:

```bash
#!/bin/bash

. /etc/rc.common

# Wait for network
CheckForNetwork
while [ "${NETWORKUP}" != "-YES-" ]; do
	sleep 5
	NETWORKUP=
	CheckForNetwork
done

datadir="$HOME/.photostream"

# Run photostream
photostream -profile linxspirationofficial

# Find new wallpaper file and update system background.
wallpaper=$(find "$datadir/cache" -iname "wallpaper_*")
/usr/bin/osascript -e "tell application \"Finder\" to set desktop picture to POSIX file \"$wallpaper\""
```

### Linux

On Linux the command to programmatically update your system wallpaper depends on your window manager and shell.

