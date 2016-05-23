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

> Tip: use *photostream* together with *cron* to update your wallpaper automatically.

