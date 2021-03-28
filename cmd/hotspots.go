package main

/*
 * Helium Analysis
 * Copyright (c) 2021 Aaron Turner  <aturner at synfin dot net>
 *
 * This program is free software: you can redistribute it
 * and/or modify it under the terms of the GNU General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or with the authors permission any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

type HotspotResponse struct {
	Data Hotspot `json:"data"`
}

type HotspotsResponse struct {
	Data   []Hotspot `json:"data"`
	Cursor string    `json:"cursor"`
}

type HotspotCache struct {
	Time     int64     `json:"time"`
	Hotspots []Hotspot `json:"hotspots"`
}

type Hotspot struct {
	Address           string       `json:"address"`
	Block             int64        `json:"block"`
	BlockAdded        int64        `json:"block_added"`
	Geocode           *GeocodeType `json:"geocode"`
	Lat               float64      `json:"lat"`
	Lng               float64      `json:"lng"`
	Location          string       `json:"location"`
	Name              string       `json:"name"`
	Nonce             int64        `json:"nonce"`
	Owner             string       `json:"owner"`
	Score             float64      `json:"score"`
	ScoreUpdateHeight int64        `json:"score_update_height"`
	Status            *StatusType  `json:"status"`
}

type StatusType struct {
	Height int64  `json:"height"`
	Online string `json:"online"`
}

const (
	HOTSPOT_CACHE_TIMEOUT = 86400 // one day
	HOTSPOT_URL           = "https://api.helium.io/v1/hotspots/%s"
	HOTSPOTS_URL          = "https://api.helium.io/v1/hotspots"
	HOTSPOT_CACHE_FILE    = "hotspots.json"
)

var HOTSPOT_CACHE map[string]Hotspot = map[string]Hotspot{}

// Looks up a hotspot by address in the cache.  If not,
// it queries the API
func getHotspot(address string) (Hotspot, error) {
	v, ok := HOTSPOT_CACHE[address]
	if ok {
		return v, nil
	} else {
		log.Debugf("cache miss: %s", address)
	}

	client := resty.New()
	resp, err := client.R().
		SetHeader("Accept", "application/json").
		SetResult(&HotspotResponse{}).
		Get(fmt.Sprintf(HOTSPOT_URL, address))

	if err != nil {
		return Hotspot{}, err
	}

	result := (resp.Result().(*HotspotResponse))
	HOTSPOT_CACHE[address] = result.Data
	return result.Data, nil
}

func getHotspotName(address string) (string, error) {
	h, err := getHotspot(address)
	if err != nil {
		return "", err
	}
	return h.Name, nil
}

func getHotspotAddress(name string) (string, error) {
	for address, hotspot := range HOTSPOT_CACHE {
		if hotspot.Name == name {
			return address, nil
		}
	}
	return "", fmt.Errorf("Unable to find %s in hotspot cache", name)
}

// Loads our hotspots from the cachefile
func loadHotspots(filename string) error {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	cache := HotspotCache{}
	err = json.Unmarshal(file, &cache)
	if err != nil {
		return err
	}

	age := time.Now().Unix() - cache.Time
	if age > HOTSPOT_CACHE_TIMEOUT {
		log.Warnf("Hotspot cache is %dhrs old.  You may want to refresh via --hotspots",
			age/60/60)
	}

	for _, v := range cache.Hotspots {
		HOTSPOT_CACHE[v.Address] = v
	}
	log.Debugf("Loaded hotspot cache")
	return nil

}

// Download hotspot data from helium.api servers and saves to filename
func downloadHotspots(filename string) error {
	hotspots := []Hotspot{}
	cursor := "" // keep track
	client := resty.New()
	first_time := true
	last_size := 0

	for first_time || cursor != "" {
		hs, c, err := getHotspotResponse(client, cursor)
		if err != nil {
			return err
		}
		log.Debugf("Retrieved %d hotspots", len(hs))
		cursor = c // keep track of the cursor for next time

		hotspots = append(hotspots, hs...)
		delta := len(hotspots) - last_size
		if delta > 250 {
			log.Infof("Loaded %d hotspots", len(hotspots))
			last_size = len(hotspots)
		}
		first_time = false
	}

	log.Debugf("found %d hotspots", len(hotspots))
	hs_cache := HotspotCache{
		Time:     time.Now().Unix(),
		Hotspots: hotspots,
	}

	jdata, err := json.MarshalIndent(hs_cache, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, jdata, 0644)
}

// Does the actual work of downloading Hotspot data
func getHotspotResponse(client *resty.Client, cursor string) ([]Hotspot, string, error) {
	var resp *resty.Response
	var err error

	if cursor == "" {
		resp, err = client.R().
			SetHeader("Accept", "application/json").
			SetResult(&HotspotsResponse{}).
			Get(HOTSPOTS_URL)
	} else {
		resp, err = client.R().
			SetHeader("Accept", "application/json").
			SetResult(&HotspotsResponse{}).
			SetQueryParams(map[string]string{
				"cursor": cursor,
			}).
			Get(HOTSPOTS_URL)
	}
	if err != nil {
		return []Hotspot{}, "", err
	}
	if resp.IsError() {
		return []Hotspot{}, "", fmt.Errorf("Error %d: %s", resp.StatusCode(), resp.String())
	}
	result := (resp.Result().(*HotspotsResponse))

	return result.Data, result.Cursor, nil
}
