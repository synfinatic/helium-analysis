package main

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
	Block             uint64       `json:"block"`
	BlockAdded        uint64       `json:"block_added"`
	Geocode           *GeocodeType `json:"geocode"`
	Lat               float64      `json:"lat"`
	Lng               float64      `json:"lng"`
	Location          string       `json:"location"`
	Name              string       `json:"name"`
	Nonce             int64        `json:"nonce"`
	Owner             string       `json:"owner"`
	Score             float64      `json:"score"`
	ScoreUpdateHeight uint64       `json:"score_update_height"`
	Status            *StatusType  `json:"status"`
}

type StatusType struct {
	Height uint64 `json:"height"`
	Online string `json:"online"`
}

const (
	HOTSPOT_URL        = "https://api.helium.io/v1/hotspots/%s"
	HOTSPOTS_URL       = "https://api.helium.io/v1/hotspots"
	HOTSPOT_CACHE_FILE = "hotspots.json"
)

var HOTSPOT_CACHE map[string]Hotspot = map[string]Hotspot{}

func getHotspot(address string) (Hotspot, error) {
	v, ok := HOTSPOT_CACHE[address]
	if ok {
		log.Debugf("cache hit:  %s => %s", address, v.Name)
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

func getHotspots() {
	file, err := ioutil.ReadFile(HOTSPOT_CACHE_FILE)
	if err == nil {
		cache := HotspotCache{}
		err = json.Unmarshal(file, &cache)
		if err == nil {
			for _, v := range cache.Hotspots {
				HOTSPOT_CACHE[v.Address] = v
			}
			log.Debugf("Loaded hotspot cache")
			return
		}
	}

	log.Fatalf("Unable to load hotspot cache")
}

// Returns a list of Challenges and the cursor location or an error
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
	result := (resp.Result().(*HotspotsResponse))

	return result.Data, result.Cursor, nil
}

// Fails to return on error
func loadHotspots(filename string) error {
	hotspots := []Hotspot{}
	cursor := "" // keep track
	client := resty.New()
	first_time := true
	last_size := 0

	log.Debugf("wel well well")
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
