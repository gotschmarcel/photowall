// Copyright 2016 Marcel Gotsch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nfnt/resize"
)

var (
	profile       string
	baseURL       string
	cacheDir      string
	bgHex         string
	outputSize    string
	outputWidth   float64
	outputHeight  float64
	gridSize      float64
	gridSpacing   float64
	gridCols      int
	outputQuality int
	setWallpaper  bool
	bgColor       color.RGBA

	wallpaperName = fmt.Sprintf("wallpaper_%d.jpg", time.Now().Unix())

	// The Instagram API doesn't allow arbitrary sizes, instead it allows:
	thumbSizes      = []float64{320.0, 360.0, 420.0, 480.0, 540.0, 640.0, 720.0, 960.0}
	urlSizeMatcher  = regexp.MustCompile("/s\\d+x\\d+/")
	urlSizeReplacer = "/s%.0fx%.0f/"
)

func init() {
	flag.StringVar(&profile, "profile", "", "User profile name")
	flag.StringVar(&baseURL, "url", "https://www.instagram.com/%s/media", "Instagram base url with profile placeholder")
	flag.StringVar(&cacheDir, "dir", "", "Cache and wallpaper directory")
	flag.StringVar(&bgHex, "bg", "FFFFFF", "Background hex color")
	flag.StringVar(&outputSize, "size", "1920x1080", "Wallpaper size")
	flag.Float64Var(&gridSpacing, "spacing", 10.0, "Space between images")
	flag.Float64Var(&gridSize, "grid", 212.0, "Grid size")
	flag.IntVar(&gridCols, "cols", 5, "Number of image columns")
	flag.IntVar(&outputQuality, "q", 90, "Output jpeg quality (1-100)")
	flag.BoolVar(&setWallpaper, "set", false, "Set system wallpaper")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -dir DIR -profile PROFILE [OPTIONS]\n", os.Args[0])
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

	fatalIf(fmt.Errorf("%q not specified", name))
}

func parseSizeOption() {
	parts := strings.Split(outputSize, "x")

	if len(parts) != 2 {
		fatalIf(fmt.Errorf("size not in format <width>x<height>"))
	}

	var (
		werr error
		herr error
	)

	outputWidth, werr = strconv.ParseFloat(parts[0], 64)
	outputHeight, herr = strconv.ParseFloat(parts[1], 64)
	if werr != nil || herr != nil {
		fatalIf(fmt.Errorf("Invalid width or height"))
	}

	if outputWidth < 0.0 || outputHeight < 0.0 {
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

	r, err := strconv.ParseInt(bgHex[0:2], 16, 0)
	fatalIf(err)

	g, err := strconv.ParseInt(bgHex[2:4], 16, 0)
	fatalIf(err)

	b, err := strconv.ParseInt(bgHex[4:6], 16, 0)
	fatalIf(err)

	bgColor.R = uint8(r)
	bgColor.G = uint8(g)
	bgColor.B = uint8(b)
	bgColor.A = 255
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

type imageInfo struct {
	URL    string  `json:"url"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type item struct {
	ID     string `json:"id"`
	Images struct {
		Thumbnail *imageInfo `json:"thumbnail"`
	} `json:"images"`
}

type media struct {
	Status string  `json:"status"`
	Items  []*item `json:"items"`
}

func fetchMediaItems() []*item {
	url := fmt.Sprintf(baseURL, profile)
	log.Printf("Fetching media from %q", url)

	resp, err := http.Get(url)
	fatalIf(err)

	defer resp.Body.Close()

	var media media
	if err := json.NewDecoder(resp.Body).Decode(&media); err != nil {
		fatalIf(fmt.Errorf("Invalid response data, the profile probably doesn't exist: %s", err.Error()))
	}

	log.Printf("Fetched %d images", len(media.Items))

	return media.Items
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

func nextThumbsize() float64 {
	// Assuming that the thumbSizes are sorted in ascending order
	for _, size := range thumbSizes {
		if size > gridSize {
			return size
		}
	}

	return thumbSizes[len(thumbSizes)-1]
}

func downloadImage(itm *item, size float64, wg *sync.WaitGroup) {
	defer wg.Done()

	log.Printf("Downloading %q", itm.ID)

	// Build thumbnail URL
	thumbSize := fmt.Sprintf(urlSizeReplacer, size, size)
	thumbURL := urlSizeMatcher.ReplaceAllString(itm.Images.Thumbnail.URL, thumbSize)

	resp, err := http.Get(thumbURL)
	if err != nil {
		log.Printf("Error: Failed to download %q, %s", thumbURL, err.Error())
		return
	}

	defer resp.Body.Close()

	// Create or truncate image file.
	imgFilePath := filepath.Join(cacheDir, itm.ID)
	file, err := os.Create(imgFilePath)
	if err != nil {
		log.Printf("Error: Failed to open file for writing %q, %s", imgFilePath, err.Error())
		return
	}

	defer file.Close()

	// Write file from response body.
	if _, err := io.Copy(file, resp.Body); err != nil {
		log.Printf("Error: Failed to write file %q, %s", imgFilePath, err.Error())
		return
	}

	log.Printf("Download of %q complete", itm.ID)
}

func downloadImages(items []*item) {
	var dls = &sync.WaitGroup{}

	thumbSize := nextThumbsize()
	log.Printf("Closest thumbnail size %f", thumbSize)

	cache := cachedImages()
	log.Printf("Found %d cached images", len(cache))

	for _, itm := range items {
		// Check if the image is cached. If it is then remove
		// it from the cache info. Anything left in the cache after
		// the loop is deprecated.
		if cache[itm.ID] {
			delete(cache, itm.ID)

			// Make sure that the image info corresponds to the cropped image
			file, err := os.Open(filepath.Join(cacheDir, itm.ID))
			fatalIf(err)
			defer file.Close()

			conf, _, err := image.DecodeConfig(file)
			fatalIf(err)

			if float64(conf.Width) == thumbSize {
				continue
			}

			// Size did not match, must download a new thumbnail.
			log.Printf("Cached image has wrong size %q - downloading new version", itm.ID)
		}

		dls.Add(1)

		go downloadImage(itm, thumbSize, dls)
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

func buildWallpaper(items []*item) {
	log.Printf("Building wallpaper based on %d images (%.0fx%.0f)", len(items), outputWidth, outputHeight)

	thumbSize := nextThumbsize()

	// Create wallpaper canvas and draw the background color.
	wp := image.NewRGBA(image.Rect(0, 0, int(outputWidth), int(outputHeight)))
	draw.Draw(wp, wp.Bounds(), &image.Uniform{bgColor}, image.ZP, draw.Src)

	// Compute number of rows and columns as well as the offset to
	// center the grid.
	//
	// Note: columns are also computed to adapt the number of columns in case
	//       that the number of items is not divisible by the number of columns
	//		 specified without a remainder.
	rows := math.Ceil(float64(len(items)) / float64(gridCols))
	cols := math.Ceil(float64(len(items)) / float64(rows))

	dx := (outputWidth - (cols*(gridSize+gridSpacing) - gridSpacing)) / 2.0
	dy := (outputHeight - (rows*(gridSize+gridSpacing) - gridSpacing)) / 2.0

	row, col := 0.0, 0.0

	// Warn if grid size exceeds canvas
	if dx < 0.0 || dy < 0.0 {
		log.Printf("Warning: grid exceeds the output size, consider specifying a smaller grid size with --grid")
	}

	for _, itm := range items {
		// Open image file
		file, err := os.Open(filepath.Join(cacheDir, itm.ID))
		fatalIf(err)

		// Decode jpeg file
		img, err := jpeg.Decode(file)
		file.Close()
		fatalIf(err)

		// Warn if upscaling is required
		if gridSize > thumbSize {
			log.Printf("Warning: Image too small %q", itm.ID)
		}

		// Resize the thumbnail image to its desired size
		// if necessary
		if img.Bounds().Dx() != int(gridSize) {
			img = resize.Resize(uint(gridSize), 0, img, resize.Lanczos3)
		}

		// Determine position in wallpaper
		cdx := dx + col*(gridSize+gridSpacing)
		cdy := dy + row*(gridSize+gridSpacing)

		// Draw scaled image onto wallpaper
		dp := image.Pt(int(cdx), int(cdy))
		bounds := image.Rectangle{dp, dp.Add(img.Bounds().Size())}
		draw.Draw(wp, bounds, img, img.Bounds().Min, draw.Src)

		// Check if column is complete
		row++
		if math.Mod(row, rows) == 0.0 {
			col++
			row = 0.0
		}
	}

	wpFile := filepath.Join(cacheDir, wallpaperName)
	file, err := os.Create(wpFile)
	fatalIf(err)

	defer file.Close()
	fatalIf(jpeg.Encode(file, wp, &jpeg.Options{Quality: outputQuality}))
}

func main() {
	flag.Parse()

	requiredOption("profile", profile)
	requiredOption("dir", cacheDir)

	parseSizeOption()
	parseBGOption()

	// Create the photo and wallpaper directory.
	createDir()

	// Request recent profile media
	items := fetchMediaItems()
	if len(items) == 0 {
		log.Printf("Nothing to do")
		return
	}

	// Download images
	downloadImages(items)

	// Create the wallpaper image composed from all downloaded images
	buildWallpaper(items)

	// Finally update the system wallpaper of the current user
	if setWallpaper {
		setSystemWallpaper()
	}

	log.Printf("Wallpaper updated")
}
