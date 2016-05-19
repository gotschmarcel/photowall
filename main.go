// Copyright 2016 Marcel Gotsch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "image/gif" // Import for support side effects only
	_ "image/png" // Import for support side effects only

	"github.com/nfnt/resize"
)

const (
	Version                  = "v1.4.0"
	InstapaperDefaultDirName = ".instapaper"
)

var (
	// Flag vars
	apiName       string
	apiKey        string
	profile       string
	tag           string
	cacheDir      string
	bgHex         string
	outputSize    string
	outputQuality int
	squareTiles   bool
	gridCols      int
	gridSize      int
	gridSpacing   int
	itemLimit     int
	showVersion   bool
	setWallpaper  bool

	// Parsed values
	outputWidth  int
	outputHeight int
	bgColor      color.RGBA

	wallpaperName = fmt.Sprintf("wallpaper_%d.jpg", time.Now().Unix())

	apiFactory = &APIFactory{make(map[string]APIFactoryFunc)}
)

type MediaItem struct {
	ID     string
	URL    string
	Width  int
	Height int
}

type API interface {
	FetchMediaItems(profile string, size int, tag string, limit int) ([]*MediaItem, error)
	SupportsOnlySquareImages() bool
}

type APIFactoryFunc func(string) API

type APIFactory struct {
	apis map[string]APIFactoryFunc
}

func (a *APIFactory) Register(name string, factoryFn APIFactoryFunc) {
	a.apis[name] = factoryFn
}

func (a *APIFactory) Create(name, key string) API {
	factoryFn := a.apis[name]

	if factoryFn == nil {
		return nil
	}

	return factoryFn(key)
}

func init() {
	flag.StringVar(&apiName, "api", "instagram", "API to use (instagram, tumblr)")
	flag.StringVar(&apiKey, "key", "", "API key")
	flag.StringVar(&profile, "profile", "", "User profile name")
	flag.StringVar(&tag, "tag", "", "Tag filter")
	flag.StringVar(&cacheDir, "dir", "", "Cache and wallpaper directory")
	flag.StringVar(&bgHex, "bg", "FFFFFF", "Background hex color")
	flag.StringVar(&outputSize, "size", "1920x1080", "Wallpaper size")
	flag.BoolVar(&squareTiles, "square", false, "Use square tiles")
	flag.IntVar(&gridSpacing, "spacing", 10.0, "Space between images")
	flag.IntVar(&gridSize, "grid", 212.0, "Grid size")
	flag.IntVar(&gridCols, "cols", 5, "Number of image columns")
	flag.IntVar(&outputQuality, "q", 90, "Output jpeg quality (1-100)")
	flag.IntVar(&itemLimit, "limit", 20, "Number of images fetched from api")
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&setWallpaper, "set", false, "Set system wallpaper")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: %s -profile PROFILE [OPTIONS]

By default instapaper stores its cached images under ~/.instapaper. If you
want to change the cache directory pass -dir <your_dir>.

Instagram:
	To use instagram pass -api instagram. The Instagram API supports
	only squared tiles and max 20 images. Since the API doesn't required
	an API token you can use it without -key. Unfortunately the tag filter
	is not available for Instagram.

Tumblr:
	To use tumblr pass -api tumblr -key api_key. This API requires an
	API token. To get an api token you must register
	an API app at https://www.tumblr.com/oauth/apps. This API supports 
	both squared and non-squared tiles. It also allows more than 20 images.

Options:
`, os.Args[0])

		flag.PrintDefaults()
	}
}

func fatalIf(err error) {
	if err == nil {
		return
	}

	log.Fatalf("Fatal: %s", err)
}

func requiredOption(name, val string) {
	if len(val) > 0 {
		return
	}

	flag.Usage()
	fatalIf(fmt.Errorf("%q not specified", name))
}

func parseSizeOption() {
	parts := strings.Split(outputSize, "x")

	if len(parts) != 2 {
		fatalIf(fmt.Errorf("size not in format <width>x<height>"))
	}

	var werr, herr error

	outputWidth, werr = strconv.Atoi(parts[0])
	outputHeight, herr = strconv.Atoi(parts[1])

	if werr != nil || herr != nil {
		fatalIf(fmt.Errorf("Invalid width or height"))
	}

	if outputWidth < 0 || outputHeight < 0 {
		fatalIf(fmt.Errorf("Size must be positive"))
	}
}

func parseBGOption() {
	// Remove leading hash
	if strings.HasPrefix(bgHex, "#") {
		bgHex = bgHex[1:]
	}

	if len(bgHex) != 6 {
		fatalIf(fmt.Errorf("Background color not in hex format"))
	}

	rgb, err := strconv.ParseInt(bgHex, 16, 0)
	fatalIf(err)

	bitMask := int64(0xFF)

	bgColor.R = uint8(rgb >> 16 & bitMask)
	bgColor.G = uint8(rgb >> 8 & bitMask)
	bgColor.B = uint8(rgb & bitMask)
	bgColor.A = 255
}

func fallbackDirOption() {
	if len(cacheDir) > 0 {
		return
	}

	usr, err := user.Current()
	if err != nil {
		fatalIf(fmt.Errorf("Unable to get home directory. Try setting -dir yourself"))
	}

	cacheDir = filepath.Join(usr.HomeDir, InstapaperDefaultDirName)
}

func createDir() {
	log.Printf("Creating data directory %q", cacheDir)
	err := os.Mkdir(cacheDir, os.ModeDir|0755)

	if os.IsExist(err) {
		log.Printf("Data directory already exists")
		return
	}

	fatalIf(err)
}

func cachedImages() map[string]bool {
	files, err := ioutil.ReadDir(cacheDir)
	fatalIf(err)

	images := make(map[string]bool)
	for _, file := range files {
		// Ignore directories
		if file.IsDir() {
			continue
		}

		images[file.Name()] = true
	}

	return images
}

func openCachedImage(id string) (image.Image, error) {
	imgFilePath := filepath.Join(cacheDir, id)

	file, err := os.Open(imgFilePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()
	return jpeg.Decode(file)
}

func cropImage(img image.Image) image.Image {
	bounds := img.Bounds()
	dx, dy := bounds.Dx(), bounds.Dy()

	ndx, ndy := dx, dy

	if dx < dy {
		ndy = dx
	} else {
		ndx = dy
	}

	offx, offy := (dx-ndx)/2, (dy-ndy)/2
	cropped := image.NewRGBA(image.Rect(0, 0, ndx, ndy))

	draw.Draw(cropped, cropped.Bounds(), img, image.Pt(offx, offy), draw.Src)

	return cropped
}

func downloadImage(item *MediaItem) {
	resp, err := http.Get(item.URL)
	if err != nil {
		log.Printf("Error: Failed to download %q, %s", item.URL, err.Error())
		return
	}

	defer resp.Body.Close()

	// Make sure it's jpeg
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		log.Printf("Error: Reading image body of %q, %s", item.URL, err.Error())
		return
	}

	// If squared tiles are requested but image isn't then crop it first.
	if squareTiles && img.Bounds().Dx() != img.Bounds().Dy() {
		img = cropImage(img)
		// Update the item information
		item.Width = img.Bounds().Dx()
		item.Height = img.Bounds().Dy()
	}

	// Create or truncate image file.
	imgFilePath := filepath.Join(cacheDir, item.ID)
	file, err := os.Create(imgFilePath)
	if err != nil {
		log.Printf("Error: Failed to open file for writing %q, %s", imgFilePath, err.Error())
		return
	}

	defer file.Close()

	if err := jpeg.Encode(file, img, &jpeg.Options{100}); err != nil {
		log.Printf("Error: Saving image %q, %s", item.URL, err.Error())
		return
	}

	log.Printf("Download of %q complete", item.ID)
}

func imageHasCorrectSize(iconf *image.Config, item *MediaItem) bool {
	if squareTiles {
		size := minInt(item.Width, item.Height)
		return iconf.Height == size && iconf.Width == size
	}

	return iconf.Width == item.Width && iconf.Height == item.Height
}

func downloadImages(items []*MediaItem) {
	var dls = &sync.WaitGroup{}

	cache := cachedImages()
	log.Printf("Found %d cached images", len(cache))

	for _, item := range items {
		// Check if the image is cached. If it is then remove
		// it from the cache info. Anything left in the cache after
		// the loop is deprecated.
		cached := cache[item.ID]

		if cached {
			delete(cache, item.ID)
		}

		dls.Add(1)

		go func(item *MediaItem, cached bool) {
			defer dls.Done()

			if cached {
				log.Printf("Checking cached image %q", item.ID)

				// Make sure that the image has the correct size and is not broken
				file, err := os.Open(filepath.Join(cacheDir, item.ID))
				if err != nil {
					log.Printf("Could not open cached version of %q, %s", item.ID, err.Error())
					goto downloadImage
				}

				defer file.Close()
				conf, _, err := image.DecodeConfig(file)
				if err != nil {
					log.Printf("Error: Could not decode jpeg header of %q", item.ID)
					goto downloadImage
				}

				if imageHasCorrectSize(&conf, item) {
					return
				}

				log.Printf("%q has wrong size", item.ID)
			}

		downloadImage:
			log.Printf("Downloading new version of %q", item.ID)
			downloadImage(item)

		}(item, cached)
	}

	dls.Wait()

	// Remove deprecated images
	for file, _ := range cache {
		imgFilePath := filepath.Join(cacheDir, file)

		log.Printf("Removing old image %q", imgFilePath)

		if err := os.Remove(imgFilePath); err != nil {
			log.Printf("Error: Failed to remove old file %q, %s", imgFilePath, err.Error())
		}
	}
}

func drawSquareGrid(wp *image.RGBA, items []*MediaItem) {
	// Compute number of rows and columns as well as the offset to
	// center the grid.
	//
	// Note: columns are also computed to adapt the number of columns in case
	//       that the number of items is not divisible by the number of columns
	//		 specified without a remainder.
	rows := ceilIntDivision(len(items), gridCols)
	cols := ceilIntDivision(len(items), rows)

	dx := (outputWidth - (cols*(gridSize+gridSpacing) - gridSpacing)) / 2
	dy := (outputHeight - (rows*(gridSize+gridSpacing) - gridSpacing)) / 2

	row, col := 0, 0

	// Warn if grid size exceeds canvas
	if dx < 0 || dy < 0 {
		log.Printf("Warning: grid exceeds the output size, consider specifying a smaller grid size with --grid")
	}

	for _, item := range items {
		img, err := openCachedImage(item.ID)
		if err != nil {
			fatalIf(fmt.Errorf("%s with image %s", err.Error(), item.ID))
		}

		// Warn if upscaling is required
		if gridSize > item.Width {
			log.Printf("Warning: Image too small %q", item.ID)
		}

		// Resize the thumbnail image to its desired size
		// if necessary
		if img.Bounds().Dx() != gridSize {
			img = resize.Resize(uint(gridSize), 0, img, resize.Lanczos3)
		}

		// Determine position in wallpaper
		cdx := dx + col*(gridSize+gridSpacing)
		cdy := dy + row*(gridSize+gridSpacing)

		// Draw scaled image onto wallpaper
		dp := image.Pt(cdx, cdy)
		bounds := image.Rectangle{dp, dp.Add(img.Bounds().Size())}
		draw.Draw(wp, bounds, img, img.Bounds().Min, draw.Src)

		// Check if column is complete
		row++
		if row == rows {
			col++
			row = 0
		}
	}

}

func drawNonSquareGrid(wp *image.RGBA, items []*MediaItem) {
	cols := gridCols
	rows := ceilIntDivision(len(items), cols)

	desiredWidth := cols*(gridSize+gridSpacing) - gridSpacing
	desiredHeights := make([]int, rows)
	aggregatedHeight := 0

	row, col := 0, 0

	// Compute row heights based on the sum of the image ratios in
	// one row and the desired width of all images in this row
	// without spacing.
	aggregatedRatio := 0.0
	for i, item := range items {
		aggregatedRatio += float64(item.Width) / float64(item.Height)
		col++

		if col == cols || i == len(items)-1 {
			rowHeight := int(float64(desiredWidth) / aggregatedRatio)
			aggregatedHeight += rowHeight
			desiredHeights[row] = rowHeight

			aggregatedRatio = 0.0
			col = 0
			row++
		}
	}

	baseDx := (outputWidth - (desiredWidth + cols*gridSpacing - gridSpacing)) / 2

	dx := baseDx
	dy := (outputHeight - (aggregatedHeight + rows*gridSpacing - gridSpacing)) / 2

	if dx < 0 || dy < 0 {
		log.Printf("Warning: grid exceeds the output size, consider specifying a smaller grid size with --grid")
	}

	desiredRowWidth := desiredWidth + (cols * gridSpacing) - gridSpacing
	rowWidth := 0
	row, col = 0, 0
	for i, item := range items {
		img, err := openCachedImage(item.ID)
		fatalIf(err)

		h := desiredHeights[row]
		w := 0 // Keep aspect ratio

		// Due to rounding errors it is possible that
		// a row may have some pixels left. Since this looks ugly
		// we need to scale the last image in a row so that
		// it fills the row completely. Even though we're
		// scaling the image not by its aspect ratio it's
		// not really visible because it's just off by a few
		// pixels.
		if col == cols-1 || i == len(items)-1 {
			aw := h * img.Bounds().Dx() / img.Bounds().Dy()
			pixLeft := desiredRowWidth - rowWidth - aw
			w = aw + pixLeft
		}

		if img.Bounds().Dy() != h {
			img = resize.Resize(uint(w), uint(h), img, resize.Lanczos3)
		}

		dp := image.Pt(dx, dy)
		bounds := image.Rectangle{dp, dp.Add(img.Bounds().Size())}
		draw.Draw(wp, bounds, img, img.Bounds().Min, draw.Src)

		dx += (img.Bounds().Dx() + gridSpacing)
		col++
		rowWidth += (img.Bounds().Dx() + gridSpacing)
		if col == cols {
			col = 0
			rowWidth = 0
			row++
			dx = baseDx
			dy += (h + gridSpacing)
		}
	}
}

func buildWallpaper(items []*MediaItem) {
	log.Printf("Building wallpaper (%s)", outputSize)

	// Create wallpaper canvas and draw the background color.
	wp := image.NewRGBA(image.Rect(0, 0, outputWidth, outputHeight))
	draw.Draw(wp, wp.Bounds(), &image.Uniform{bgColor}, image.ZP, draw.Src)

	// Choose drawing algorithm
	if squareTiles {
		drawSquareGrid(wp, items)
	} else {
		drawNonSquareGrid(wp, items)
	}

	wpFile := filepath.Join(cacheDir, wallpaperName)
	file, err := os.Create(wpFile)
	fatalIf(err)

	defer file.Close()
	fatalIf(jpeg.Encode(file, wp, &jpeg.Options{Quality: outputQuality}))
}

func main() {
	flag.Parse()

	// Check version flag
	if showVersion {
		fmt.Println(Version)
		return
	}

	requiredOption("profile", profile)

	parseSizeOption()
	parseBGOption()
	fallbackDirOption()

	api := apiFactory.Create(apiName, apiKey)
	if api == nil {
		fatalIf(fmt.Errorf("%q API not supported", apiName))
	}

	// Check if the api supports non-square tiles
	if !squareTiles && api.SupportsOnlySquareImages() {
		log.Printf("The %q API supports only square tiles - falling back", apiName)
		squareTiles = true
	}

	// Create the photo and wallpaper directory.
	createDir()

	// Request recent profile media
	items, err := api.FetchMediaItems(profile, gridSize, tag, itemLimit)
	fatalIf(err)

	if l := len(items); l == 0 {
		log.Printf("Nothing to do")
		return
	} else {
		log.Printf("Fetched %d media items", l)
	}

	// Download images
	downloadImages(items)

	// Create the wallpaper image composed from all downloaded images
	buildWallpaper(items)

	// Finally update the system wallpaper of the current user
	if setWallpaper {
		log.Printf("Setting system wallpaper")

		wpFile := filepath.Join(cacheDir, wallpaperName)
		wpFile, err := filepath.Abs(wpFile)
		fatalIf(err)

		cmd := setSystemWallpaperCmd(wpFile)
		fatalIf(cmd.Run())
	}
}
