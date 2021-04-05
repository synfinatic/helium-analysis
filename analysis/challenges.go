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
	"math"
	"time"

	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
)

type ChallengeResponse struct {
	Data   []Challenges `json:"data"`
	Cursor string       `json:"cursor"`
}

type ChallengeCache struct {
	CacheTime  int64        `json:"cache_time"`
	Address    string       `json:"address"`
	StartDate  int64        `json:"start_date"`
	Challenges []Challenges `json:"challenges"`
}

type Challenges struct {
	Type               string      `json:"type"`
	Time               int64       `json:"time"`
	Secret             string      `json:"secret"`
	Path               *[]PathType `json:"path"`
	OnionKeyHash       string      `json:"onion_key_hash"`
	Height             int64       `json:"height"`
	Hash               string      `json:"hash"`
	Fee                int         `json:"fee"`
	ChallengerOwner    string      `json:"challenger_owner"`
	ChallengerLon      float64     `json:"challenger_lon"`
	ChallengerLat      float64     `json:"challenger_lat"`
	ChallengerLocation string      `json:"challenger_location"`
	Challenger         string      `json:"challenger"`
}

type PathType struct {
	Witnesses          *[]WitnessType `json:"witnesses"`
	Receipt            *ReceiptType   `json:"receipt"`
	Geocode            *GeocodeType   `json:"geocode"`
	ChallengeeOwner    string         `json:"challengee_owner"`
	ChallengeeLon      float64        `json:"challengee_lon"`
	ChallengeeLat      float64        `json:"challengee_lat"`
	ChallengeeLocation string         `json:"challengee_location"`
	Challengee         string         `json:"challengee"`
}

type WitnessType struct {
	Timestamp  int64   `json:"timestamp"`
	Signal     int     `json:"signal"`
	PacketHash string  `json:"packet_hash"`
	Owner      string  `json:"owner"`
	Location   string  `json:"location"`
	Gateway    string  `json:"gateway"`
	Snr        float64 `json:"snr"`
	IsValid    bool    `json:"is_valid"`
	// Available fields we don't need
	//	Frequency          float64     `json:"frequency"`
	//	Datarate           string      `json:"datarate"`
	//	Channel            int         `json:"channel"`
}

type ReceiptType struct {
	Timestamp int64  `json:"timestamp"`
	Signal    int    `json:"signal"`
	Origin    string `json:"origin"`
	Gateway   string `json:"gateway"`
	Data      string `json:"data"`
}

type GeocodeType struct {
	ShortStreet  string `json:"short_street"`
	ShortState   string `json:"short_state"`
	ShortCountry string `json:"short_country"`
	ShortCity    string `json:"short_city"`
	Street       string `json:"long_street"`
	State        string `json:"long_state"`
	Country      string `json:"long_country"`
	City         string `json:"long_city"`
}

type ChallengeResult struct {
	Timestamp int64
	Address   string
	Signal    int
	Location  string
}

type WitnessResult struct {
	Timestamp      int64   `json:"timestamp"`
	Address        string  `json:"address"`
	Witness        string  `json:"witness"`
	Type           RXTX    `json:"type"`
	Signal         int     `json:"signal"`
	Valid          bool    `json:"valid"`
	Km             float64 `json:"km"`
	Mi             float64 `json:"mi"`
	Location       string  `json:"location"`
	Snr            float64 `json:"snr"`
	ValidThreshold float64 `json:"valid_threshold"`
	Hash           string  `json:"hash"`
}

const CHALLENGE_URL = "https://api.helium.io/v1/hotspots/%s/challenges"

func getTxResults(address string, challenges []Challenges) ([]ChallengeResult, error) {
	results := []ChallengeResult{}
	for _, entry := range challenges {
		if entry.Type != "poc_receipts_v1" {
			log.Warnf("unexpected entry type: %s", entry.Type)
			continue
		}

		for _, path := range *entry.Path {
			if path.Challengee != address {
				// challengee's send the PoC
				continue
			}
			for _, witness := range *path.Witnesses {
				results = append(results, ChallengeResult{
					Address:   witness.Gateway,
					Timestamp: witness.Timestamp,
					Signal:    witness.Signal,
					Location:  witness.Location,
				})
			}
		}
	}
	log.Debugf("found %d Tx results for %s", len(results), address)
	return results, nil
}

func getRxResults(address string, challenges []Challenges) ([]ChallengeResult, error) {
	results := []ChallengeResult{}
	for _, entry := range challenges {
		if entry.Type != "poc_receipts_v1" {
			log.Warnf("unexpected entry type: %s", entry.Type)
			continue
		}

		for _, path := range *entry.Path {
			if path.Challengee == address {
				// challengee's receive the PoC
				continue
			}
			for _, witness := range *path.Witnesses {
				results = append(results, ChallengeResult{
					Address:   witness.Gateway,
					Timestamp: witness.Timestamp,
					Signal:    witness.Signal,
					Location:  witness.Location,
				})
			}
		}
	}
	log.Debugf("found %d Rx results for %s", len(results), address)
	return results, nil
}

func (b *BoltDB) getWitnessResults(address, witness string, challenges []Challenges) ([]WitnessResult, error) {
	results := []WitnessResult{}
	aHost, err := b.GetHotspot(address)
	if err != nil {
		return []WitnessResult{}, err
	}

	for _, entry := range challenges {
		if entry.Type != "poc_receipts_v1" {
			log.Warnf("unexpected entry type: %s", entry.Type)
			continue
		}
		for _, path := range *entry.Path {
			var rxtx RXTX = RX
			if path.Challengee == address {
				rxtx = TX
			}

			for _, wit := range *path.Witnesses {
				// ignore witness for beacons not sent by us or when we're not the challengee
				if address == wit.Gateway && path.Challengee != witness {
					continue
				} else if rxtx == TX && wit.Gateway != witness {
					continue
				} else if rxtx == RX && wit.Gateway != address {
					continue
				}

				wHost, err := b.GetHotspot(witness)
				if err != nil {
					log.WithError(err).Errorf("Unable to lookup: %s", witness)
					continue
				}

				km, mi, err := getDistance(aHost, wHost)
				if err != nil {
					km = 0.0
					mi = 0.0
				}

				results = append(results, WitnessResult{
					Timestamp:      int64(wit.Timestamp),
					Address:        address,
					Witness:        wit.Gateway,
					Signal:         wit.Signal,
					Type:           rxtx,
					Valid:          wit.IsValid,
					Snr:            wit.Snr,
					ValidThreshold: minRssiPerSnr(wit.Snr),
					Km:             km,
					Mi:             mi,
					Location:       wit.Location,
					Hash:           entry.Hash,
				})
			}
		}
	}
	log.Debugf("found %d witness results for %s<->%s", len(results), address, witness)
	return results, nil
}

// returns a unique list of addresses seen in the challenge results
func getAddresses(results []ChallengeResult) ([]string, error) {
	addrs := map[string]int{}
	for _, result := range results {
		_, ok := addrs[result.Address]
		if !ok {
			addrs[result.Address] = 1
		} else {
			addrs[result.Address] += 1
		}
	}
	ret := []string{}
	for k, _ := range addrs {
		ret = append(ret, k)
	}
	return ret, nil
}

// returns lists of timestamps and signal values
func getSignalsTimeSeriesChallenge(address string, results []ChallengeResult) ([]time.Time, []float64) {
	tss := []time.Time{}
	signals := []float64{}
	for _, result := range results {
		if result.Address == address {
			// tss = append(tss, float64(result.Timestamp))
			tss = append(tss, time.Unix(int64(result.Timestamp/1000000000), 0))
			signals = append(signals, float64(result.Signal))
		}
	}
	return tss, signals
}

func getSignalsSeriesChallenge(address string, results []ChallengeResult) ([]float64, []float64) {
	tss := []float64{}
	signals := []float64{}
	for _, result := range results {
		if result.Address == address {
			tss = append(tss, float64(result.Timestamp))
			signals = append(signals, float64(result.Signal))
		}
	}
	return tss, signals
}

// generates a ton of output
func printChallenges(challenges []Challenges) {
	fmt.Printf(spew.Sdump(challenges))
}

// write the cache file
func WriteChallenges(challenges []Challenges, filename, address string, start time.Time) error {
	cache := ChallengeCache{
		CacheTime:  time.Now().UTC().Unix(),
		Address:    address,
		StartDate:  start.Unix(),
		Challenges: challenges,
	}
	b, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, b, 0644)
}

// read the cache file
func LoadChallenges(filename, address string, expires int64, start time.Time, forceCache bool) ([]Challenges, error) {
	cache := ChallengeCache{}
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return []Challenges{}, err
	}
	err = json.Unmarshal(bytes, &cache)

	if err != nil {
		return []Challenges{}, err
	}
	if !forceCache && cache.CacheTime+expires < time.Now().UTC().Unix() {
		return []Challenges{}, fmt.Errorf("Challenge cache is old.")
	}
	recordTime := time.Unix(cache.StartDate, 0)
	if start.Before(recordTime) {
		return []Challenges{}, fmt.Errorf("Challenge cache wrong amount of time.")
	}
	if cache.Address != address {
		return []Challenges{}, fmt.Errorf("Challenge cache is for different hotspot.")
	}

	// need to remove any records older than start time
	challenges := []Challenges{}
	for _, c := range cache.Challenges {
		recordTime, err = c.GetTime()
		if err != nil {
			return []Challenges{}, err
		}
		if start.Before(recordTime) {
			challenges = append(challenges, c)
		}
	}

	return challenges, nil
}

// returns the highest nanosec block time less than or equal to the given height
func getTimeForHeight(height int64, challenges []Challenges) (int64, error) {
	var t_height int64 = math.MaxInt64
	var t int64 = 0
	var err error
	for _, c := range challenges {
		if c.Height <= t_height && c.Height > height {
			t, err = c.GetTimestamp()
			if err != nil {
				continue
			}
			t_height = c.Height
		}
	}
	if t > 0 {
		return t, nil
	}
	return 0, fmt.Errorf("Unable to find time for height %d", height)
}

// Tries to figure out the Timestamp for the given challenge
func (c *Challenges) GetTimestamp() (int64, error) {
	if c.Path == nil {
		return 0, fmt.Errorf("No paths: unable to determine timestamp for %s@%d",
			c.Type, c.Time)
	}
	p := *c.Path

	if p[0].Receipt != nil {
		r := p[0].Receipt
		return r.Timestamp, nil
	}

	if p[0].Witnesses != nil {
		w := *p[0].Witnesses
		return w[0].Timestamp, nil
	}

	return 0, fmt.Errorf("No data: unable to determine timestamp for %s@%d",
		c.Type, c.Time)
}

// Tries to figure out the Time for the given challenge
func (c *Challenges) GetTime() (time.Time, error) {
	t, err := c.GetTimestamp()
	if err != nil {
		return time.Time{}, err
	}
	secs := t / 1000000000
	nsec := t - (secs * 1000000000)
	return time.Unix(secs, nsec), nil
}
