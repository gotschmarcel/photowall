# Instapaper

Instapaper generates a wallpaper based on recent images of an *Instagram* profile.

## Build

To build *Instapaper* simply use `go build -tags osx` with an additional system build tag.
Updating the system wallpaper is different for each operating system and even the desktop environments, e.g. Gnome, thus
*Instapaper* provides update functions for different systems. Specify your system using one of the following build tags:

* osx (Mac OS X)
* gnome (Linux, Gnome)

## Usage

The simplest call must specify the data directory, where *Instapaper* stores its cached images and the resulting wallpaper, as well as
the *Instagram* profile, from which the media will be downloaded.

```bash
$ instapaper -dir "~/.instapaper" -profile "jondoe"
```

*Instapaper* provides a lot more options, such as background color, output size, spacing and more. To
get an overview of all available options run `$ instapaper -h`

> Tip: use *instapaper* together with *cron* to update your wallpaper automatically.

