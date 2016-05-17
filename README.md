# Instapaper

Instapaper generates a wallpaper based on recent images of an *Instagram* profile.

## Build & Install

Install *Instapaper* with:

```bash
$ go get -tags <system> github.com/gotschmarcel/instapaper
```

The system placeholder must be filled with one of the following tags, specifying the OS and desktop environment:

* osx (Mac OS X)
* lnx_gnome2 (Linux, Gnome 2)
* lnx_gnome3 (Linux, Gnome 3)

## Usage

The simplest call must specify the data directory, where *Instapaper* stores its cached images and the resulting wallpaper, as well as
the *Instagram* profile, from which the media will be downloaded.

```bash
$ instapaper -dir "~/.instapaper" -profile "jondoe"
```

It is also possible to automatically update the system wallpaper by using `-set`:

```bash
$ instapaper -dir "~/.instapaper" -profile "jondoe" -set
```

*Instapaper* provides a lot more options, such as background color, output size, spacing and more. To
get an overview of all available options run `$ instapaper -h`

> Tip: use *instapaper* together with *cron* to update your wallpaper automatically.

