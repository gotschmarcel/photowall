// Copyright 2016 Marcel Gotsch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const TumblrPageSize = 20

type TumblrAPI struct {
	Key     string
	BaseURL string
}

func (ta *TumblrAPI) FetchMediaItems(profile string, size int, tag string, limit int) ([]*MediaItem, error) {
	pages := ceilIntDivision(limit, TumblrPageSize)
	pageSize := TumblrPageSize
	var items []*MediaItem

	profileURL, err := url.Parse(fmt.Sprintf(ta.BaseURL, profile))
	if err != nil {
		return nil, err
	}

	q := profileURL.Query()

	// Set authentication key.
	q.Set("api_key", ta.Key)

	// Set tag filter if specified.
	if len(tag) > 0 {
		q.Set("tag", tag)
	}

	for p := 0; p < pages; p++ {
		if limit < TumblrPageSize {
			pageSize = limit
		}

		q.Set("offset", strconv.Itoa(p*TumblrPageSize))
		q.Set("limit", strconv.Itoa(pageSize))

		profileURL.RawQuery = q.Encode()

		itms, err := ta.fetchItemsForPage(profileURL.String(), size)
		if err != nil {
			return nil, err
		}

		// API sources drained
		if len(itms) == 0 {
			break
		}

		items = append(items, itms...)
		limit -= len(itms)
	}

	return items, nil
}

func (ta *TumblrAPI) fetchItemsForPage(endPoint string, size int) ([]*MediaItem, error) {
	resp, err := http.Get(endPoint)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errInfo struct {
			Meta *struct {
				Msg string `json:"msg"`
			} `json:"meta"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&errInfo); err != nil {
			return nil, err
		}

		return nil, fmt.Errorf(errInfo.Meta.Msg)
	}

	var media struct {
		Response *struct {
			Posts []*struct {
				ID     int `json:"id"`
				Photos []*struct {
					AltSizes []*struct {
						URL    string `json:"url"`
						Width  int    `json:"width"`
						Height int    `json:"height"`
					} `json:"alt_sizes"`
					OriginalSize *struct {
						URL    string `json:"url"`
						Width  int    `json:"width"`
						Height int    `json:"height"`
					} `json:"original_size"`
				} `json:"photos"`
			} `json:"posts"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&media); err != nil {
		return nil, err
	}

	// Prealloc mediaItems
	mediaItems := make([]*MediaItem, 0, len(media.Response.Posts))

	for _, post := range media.Response.Posts {
		item := &MediaItem{}
		item.ID = strconv.Itoa(post.ID)

		photo := post.Photos[0]
		sizeInfo := photo.OriginalSize

		// Search for smaller versions
		for _, s := range photo.AltSizes {
			if s.Width < size || s.Height < size {
				break
			}

			sizeInfo = s
		}

		item.URL = sizeInfo.URL
		item.Width = sizeInfo.Width
		item.Height = sizeInfo.Height

		mediaItems = append(mediaItems, item)
	}

	return mediaItems, nil
}

func (ta *TumblrAPI) SupportsOnlySquareImages() bool {
	return false
}

func NewTumblrAPI(key string) API {
	return &TumblrAPI{key, "https://api.tumblr.com/v2/blog/%s/posts/photo"}
}

func init() {
	apiFactory.Register("tumblr", NewTumblrAPI)
}
