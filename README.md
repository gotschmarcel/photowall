# PhotoStream

Generate grid wallpapers based on photo streams from *Instagram, Tumblr or 500px*.

## Build & Install

Install *photostream* with:

```bash
$ go get github.com/gotschmarcel/photostream
```

## Usage

The only required option is `-profile`. It specifies the account name from which the photos should be taken. The default
API is *Instagram*.

```bash
$ photostream -profile jondoe
```

*photostream* provides a lot of options, such as background color, output size, spacing and more. To
get an overview of all available options run `$ photostream -h`

### Instagram

The Instagram sandbox allows you to use up to 20 square photos. Non-square photos are not supported! Also the
`-tag` option is not available.

Example:

```bash
$ photostream -profile linxspirationofficial
```

### Tumblr

In order to use the Tumblr API you must register your own application at <https://www.tumblr.com/oauth/apps>.
After that, use the **Consumer Key** with `-key <consumer_key>` and you're good to go. Tumblr allows you to use unlimited photos as well as the **tag** filter.

Example:

```bash
$ photostream -api tumblr -key my_consumer_key -profile linxspiration.com -tags architecture
```

### 500px

500px requires a **Consumer Key**, so you must register an app at <https://500px.com> under `Settings/Applications`.
Use `-key <consumer_key>` to pass it to *photostream*. 500px allows you to use as unlimited photos. Also the **tag** filter
can be used to filter specific [**categories**](https://github.com/500px/api-documentation/blob/master/basics/formats_and_terms.md#categories).
The `-profile` options is a bit more complex with 500px, so take a look at the [global features](https://github.com/500px/api-documentation/blob/master/endpoints/photo/GET_photos.md#global-features) section. To use only a single user pass `-profile "user:<username>"`.

Example:

```bash
$ photostream -api "500px" -key my_consumer_key -profile popular -tags "Black and White,Animals"
```

or

```bash
$ photostream -api "500px" -key my_consumer_key -profile user:mataneshel -tags "Black and White"
```

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

