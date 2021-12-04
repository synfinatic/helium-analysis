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
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

const (
	HOTSPOT_URL    = "https://api.helium.io/v1/hotspots/%s"
	HOTSPOTS_URL   = "https://api.helium.io/v1/hotspots"
	HEIGHT_URL     = "https://api.helium.io/v1/blocks/height"
	CHALLENGE_URL  = "https://api.helium.io/v1/hotspots/%s/challenges"
	RETRY_ATTEMPTS = 10
)

type HotspotResponse struct {
	Data Hotspot `json:"data"`
}

type HotspotsResponse struct {
	Data   []Hotspot `json:"data"`
	Cursor string    `json:"cursor"`
}

type HeightResponse struct {
	Data map[string]int64 `json:"data"`
}

type TooBusyError struct {
	Error    string        `json:"error"`
	ComeBack time.Duration `json:"come_back_in_ms"`
}

// retry function for 429 errors from Helium API servers
func heliumApiRetry(client *resty.Client, resp *resty.Response) (time.Duration, error) {
	if resp.StatusCode() == http.StatusTooManyRequests {
		tooBusy := TooBusyError{}
		if err := json.Unmarshal([]byte(resp.String()), &tooBusy); err != nil {
			return 0, err
		}
		if tooBusy.Error == "Too Busy" {
			log.Infof("Server is too busy. Asked to wait %dms.", tooBusy.ComeBack)
			return time.Duration(tooBusy.ComeBack / 10 * time.Millisecond), nil
		}
	}
	log.Errorf("Using default 5sec delay.  error was: %d: %s", resp.StatusCode(), resp.String())
	return time.Duration(5 * time.Second), nil
}

func NewRestyClient() *resty.Client {
	client := resty.New()
	client.SetRetryCount(3).SetRetryAfter(heliumApiRetry).AddRetryCondition(
		func(r *resty.Response, err error) bool {
			return r.StatusCode() == http.StatusTooManyRequests // 429
		},
	).SetRetryMaxWaitTime(time.Duration(10 * time.Second))
	return client
}

// Gets the current height of the blockchain
func GetCurrentHeight() (int64, error) {
	var resp *resty.Response
	var err error

	client := NewRestyClient()
	resp, err = client.R().
		SetHeader("Accept", "application/json").
		SetResult(&HeightResponse{}).
		Get(HEIGHT_URL)

	if err != nil {
		return 0, err
	}
	if resp.IsError() {
		return 0, fmt.Errorf("Error %d: %s", resp.StatusCode(), resp.String())
	}
	result := (resp.Result().(*HeightResponse))
	if val, ok := result.Data["height"]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("Missing height in API reponse")
}

// Download hotspot data from helium.api servers
func FetchHotspots() ([]Hotspot, error) {
	hotspots := []Hotspot{}
	cursor := "" // keep track
	first_time := true
	last_size := 0

	client := NewRestyClient()

	for first_time || cursor != "" {
		hs, c, err := getHotspotResponse(client, cursor)
		if err != nil {
			return []Hotspot{}, err
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
		time.Sleep(time.Duration(250) * time.Millisecond) // sleep 250ms between calls
	}

	log.Debugf("found %d hotspots", len(hotspots))
	return hotspots, nil
}

// Download all the challenges from the API.  Returns newest first (LIFO)
func FetchChallenges(address string, start time.Time) ([]Challenges, error) {
	challenges := []Challenges{}
	totalChallenges := 0
	cursor := "" // keep track
	loadMoreRecords := true
	attempt := 0

	client := NewRestyClient()

	for loadMoreRecords {
		chals, c, err := getChallengeResponse(client, address, cursor)
		if err != nil && attempt < RETRY_ATTEMPTS {
			attempt += 1
			log.Errorf("Error from server.  Backing off attempt %d and trying again...", attempt)
			log.Debugf("%s", err)
			time.Sleep(time.Duration(1500*attempt) * time.Millisecond) // back off 1.5 secs
			continue
		} else if err != nil {
			return []Challenges{}, fmt.Errorf("Unable to load challenges: %s", err)
		} else if totalChallenges == 0 && len(chals) == 0 && c == "" {
			// sometimes we get 0 results, but a cursor for "more"
			return []Challenges{}, fmt.Errorf("0 challenges fetched for %s.  Invalid address?", address)
		} else if len(chals) == 0 && c == "" {
			log.Warnf("Only able to retrieve %d challenges", totalChallenges)
			return challenges, nil
		}

		cursor = c // keep track of the cursor for next time

		for i := 0; i < len(chals); i++ {
			challengeTime, err := chals[i].GetTime()
			if err != nil {
				return []Challenges{}, err
			}
			if challengeTime.Before(start) {
				loadMoreRecords = false
				break
			}

			challenges = append(challenges, chals[i])
			totalChallenges += 1
			if totalChallenges%100 == 0 {
				log.Infof("Loaded %d challenges", totalChallenges)
				t, err := chals[i].GetTime()
				if err != nil {
					log.WithError(err).Errorf("Unable to determine time")
				} else {
					log.Infof("Last challenge time: %s", t.Format(TIME_FORMAT))
				}
			}
			time.Sleep(time.Duration(750) * time.Millisecond) // sleep 750ms between calls
		}
	}

	log.Debugf("found %d challenges for %s", len(challenges), address)
	return challenges, nil
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
		log.Debugf("Helium API Cursor: %s", cursor)
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

// Returns a list of Challenges and the cursor location or an error
func getChallengeResponse(client *resty.Client, address string, cursor string) ([]Challenges, string, error) {
	var resp *resty.Response
	var err error

	if cursor == "" {
		resp, err = client.R().
			SetHeader("Accept", "application/json").
			SetResult(&ChallengeResponse{}).
			Get(fmt.Sprintf(CHALLENGE_URL, address))
	} else {
		log.Debugf("Helium API Cursor: %s", cursor)
		resp, err = client.R().
			SetHeader("Accept", "application/json").
			SetResult(&ChallengeResponse{}).
			SetQueryParams(map[string]string{
				"cursor": cursor,
			}).
			Get(fmt.Sprintf(CHALLENGE_URL, address))
	}
	if err != nil {
		return []Challenges{}, "", err
	}
	if resp.IsError() {
		return []Challenges{}, "", fmt.Errorf("Error %d: %s", resp.StatusCode(), resp.String())
	}
	result := (resp.Result().(*ChallengeResponse))

	return result.Data, result.Cursor, nil
}
