// Copyright 2016 Marcel Gotsch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
)

const InstagramMediaLimit = 20

type InstagramAPI struct {
	BaseURL     string
	thumbSizes  []int
	urlSizePart *regexp.Regexp
	urlSizeTpl  string
}

func (ia *InstagramAPI) FetchMediaItems(options APIFetchOptions) ([]*MediaItem, error) {
	profileURL := fmt.Sprintf(ia.BaseURL, options.Profile)

	resp, err := http.Get(profileURL)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var media struct {
		Items []*struct {
			ID     string `json:"id"`
			Images *struct {
				Thumbnail *struct {
					URL string `json:"url"`
				} `json:"thumbnail"`
			} `json:"images"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&media); err != nil {
		return nil, err
	}

	bestSize := ia.findBestSize(options.Size)
	bestSizeURLPart := fmt.Sprintf(ia.urlSizeTpl, bestSize, bestSize)

	if options.Limit > InstagramMediaLimit {
		return nil, fmt.Errorf("Instagram supporty only %d photos, limit too high", InstagramMediaLimit)
	}

	mediaItems := make([]*MediaItem, 0, options.Limit)

	for _, item := range media.Items[:options.Limit] {
		mediaURL := ia.urlSizePart.ReplaceAllString(item.Images.Thumbnail.URL, bestSizeURLPart)
		mediaItems = append(mediaItems, &MediaItem{item.ID, mediaURL, bestSize, bestSize})
	}

	return mediaItems, nil
}

func (ia *InstagramAPI) SupportsOnlySquareImages() bool {
	return true
}

func (ia *InstagramAPI) findBestSize(size int) int {
	// Assuming that the thumbSizes are sorted in ascending order
	for _, s := range ia.thumbSizes {
		if s > size {
			return s
		}
	}

	return ia.thumbSizes[len(ia.thumbSizes)-1]
}

func NewInstagramAPI(string) API {
	return &InstagramAPI{
		BaseURL:     "https://instagram.com/%s/media",
		thumbSizes:  []int{320, 360, 420, 480, 540, 640, 720, 960},
		urlSizePart: regexp.MustCompile("/s\\d+x\\d+/"),
		urlSizeTpl:  "/s%dx%d/",
	}
}

func init() {
	apiFactory.Register("instagram", NewInstagramAPI)
}
