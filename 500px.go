// Copyright 2016 Marcel Gotsch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	FiveHundredPxPageSize = 20
)

var (
	FiveHundredPxSquareSizes = map[string]int{
		"1":   70,
		"2":   140,
		"3":   280,
		"100": 100,
		"200": 200,
		"440": 440,
		"600": 600,
	}

	FiveHundredPxSizes = map[string]int{
		"4":    900,
		"5":    1170,
		"30":   256,
		"1080": 1080,
		"1600": 1600,
		"2048": 2048,
	}
)

type FiveHundredPxAPI struct {
	Key     string
	BaseURL string
}

func (fa *FiveHundredPxAPI) FetchMediaItems(options APIFetchOptions) ([]*MediaItem, error) {
	profileParts := strings.Split(options.Profile, ":")
	feature := profileParts[0]

	profileURL, err := url.Parse(fa.BaseURL)
	if err != nil {
		return nil, err
	}

	q := profileURL.Query()

	q.Set("consumer_key", fa.Key)
	q.Set("feature", feature)

	if feature == "user" {
		if len(profileParts) != 2 {
			return nil, fmt.Errorf("Missing username in profile - user:<username>")
		}

		q.Set("username", profileParts[1])
	}

	if len(options.Tag) > 0 {
		q.Set("only", options.Tag)
	}

	sizeID, size := fa.findBestSize(options.Size, options.Square)
	if len(sizeID) == 0 {
		return nil, fmt.Errorf("500px doesn't support image size %d", options.Size)
	}
	q.Set("image_size", sizeID)

	limit := options.Limit
	pages := ceilIntDivision(limit, FiveHundredPxPageSize)
	items := make([]*MediaItem, 0, limit)

	for page := 1; page <= pages; page++ {
		q.Set("page", strconv.Itoa(page))
		q.Set("rpp", strconv.Itoa(FiveHundredPxPageSize))

		profileURL.RawQuery = q.Encode()

		pageItems, err := fa.fetchItemsForPage(profileURL.String(), size, options.Square)
		if err != nil {
			return nil, err
		}

		// API sources drained.
		if len(pageItems) == 0 {
			break
		}

		limit -= len(pageItems)

		// Remove any items over limit
		if limit < 0 {
			pageItems = pageItems[:len(pageItems)+limit]
		}

		items = append(items, pageItems...)
	}

	return items, nil
}

func (fa *FiveHundredPxAPI) SupportsOnlySquareImages() bool {
	return false
}

func (fa *FiveHundredPxAPI) findBestSize(size int, square bool) (string, int) {
	availableSizes := FiveHundredPxSizes

	if square {
		availableSizes = FiveHundredPxSquareSizes
	}

	lastDiff := math.MaxInt32
	var bestID string
	var bestSize int

	for id, s := range availableSizes {
		if diff := s - size; diff >= 0 && diff < lastDiff {
			lastDiff = diff
			bestID = id
			bestSize = s
		}
	}

	return bestID, bestSize
}

func (fa *FiveHundredPxAPI) fetchItemsForPage(url string, size int, square bool) ([]*MediaItem, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errInfo struct {
			Error string `json:"error"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&errInfo); err != nil {
			return nil, err
		}

		return nil, fmt.Errorf(errInfo.Error)
	}

	var media struct {
		Photos []*struct {
			ID     int `json:"id"`
			Width  int `json:"width"`
			Height int `json:"height"`
			Images []*struct {
				URL string `json:"url"`
			} `json:"images"`
		} `json:"photos"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&media); err != nil {
		return nil, err
	}

	// Preallocate items slice
	mediaItems := make([]*MediaItem, len(media.Photos))

	for i, photo := range media.Photos {
		item := &MediaItem{ID: strconv.Itoa(photo.ID), URL: photo.Images[0].URL}

		if square {
			item.Width = size
			item.Height = size
		} else {
			// Photo has original size, so we must determine the scale
			// by dividing the new size by the longest edge and then
			// scale the original size.
			ratio := float64(size) / float64(maxInt(photo.Width, photo.Height))

			item.Width = int(ratio * float64(photo.Width))
			item.Height = int(ratio * float64(photo.Height))
		}

		mediaItems[i] = item
	}

	return mediaItems, nil
}

func NewFiveHundredPxAPI(key string) API {
	return &FiveHundredPxAPI{key, "https://api.500px.com/v1/photos"}
}

func init() {
	apiFactory.Register("500px", NewFiveHundredPxAPI)
}
