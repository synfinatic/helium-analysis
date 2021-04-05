package analysis

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

type HotspotCache struct {
	Time     int64     `json:"time"`
	Hotspots []Hotspot `json:"hotspots"`
}

type Hotspot struct {
	Address     string       `json:"address"`
	Block       int64        `json:"block"`
	BlockAdded  int64        `json:"block_added"`
	Geocode     *GeocodeType `json:"geocode"`
	Lat         float64      `json:"lat"`
	Lng         float64      `json:"lng"`
	Location    string       `json:"location"`
	Name        string       `json:"name"`
	Nonce       int64        `json:"nonce"`
	Owner       string       `json:"owner"`
	RewardScale float64      `json:"reward_scale"`
	Status      *StatusType  `json:"status"`
}

type StatusType struct {
	Height int64  `json:"height"`
	Online string `json:"online"`
}

const (
	HOTSPOT_CACHE_TIMEOUT = 86400 // one day
	HOTSPOT_CACHE_FILE    = "hotspots.json"
)

var HOTSPOT_CACHE map[string]Hotspot = map[string]Hotspot{}

// Looks up a hotspot by address in the cache.  If not,
// it queries the API
func GetHotspot(address string) (Hotspot, error) {
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

func GetHotspotName(address string) (string, error) {
	h, err := GetHotspot(address)
	if err != nil {
		return "", err
	}
	return h.Name, nil
}

func GetHotspotAddress(name string) (string, error) {
	for address, hotspot := range HOTSPOT_CACHE {
		if hotspot.Name == name {
			return address, nil
		}
	}
	return "", fmt.Errorf("Unable to find %s in hotspot cache", name)
}

// Loads our hotspots from the cachefile returns true if should be refreshed
func LoadHotspots(filename string) (error, bool) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return err, false
	}
	cache := HotspotCache{}
	err = json.Unmarshal(file, &cache)
	if err != nil {
		return err, false
	}

	age := time.Now().UTC().Unix() - cache.Time
	tooOld := false
	if age > HOTSPOT_CACHE_TIMEOUT {
		log.Warnf("Hotspot cache is %dhrs old.", age/60/60)
		tooOld = true
	}

	for _, v := range cache.Hotspots {
		HOTSPOT_CACHE[v.Address] = v
	}
	log.Debugf("Loaded hotspot cache")
	return nil, tooOld

}
