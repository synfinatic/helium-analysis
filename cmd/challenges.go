package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"github.com/umahmood/haversine"
)

type ChallengeResponse struct {
	Data   []Challenges `json:"data"`
	Cursor string       `json:"cursor"`
}

type ChallengeCache struct {
	Time       int64        `json:"time"`
	Address    string       `json:"address"`
	Count      int          `json:"count"`
	Challenges []Challenges `json:"challenges"`
}

type Challenges struct {
	Type               string      `json:"type"`
	Time               uint32      `json:"time"`
	Secret             string      `json:"secret"`
	Path               *[]PathType `json:"path"`
	OnionKeyHash       string      `json:"onion_key_hash"`
	Height             int         `json:"height"`
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
	Timestamp  int64  `json:"timestamp"`
	Signal     int    `json:"signal"`
	PacketHash string `json:"packet_hash"`
	Owner      string `json:"owner"`
	Location   string `json:"location"`
	Gateway    string `json:"gateway"`
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
	Timestamp int64
	Address   string
	Witness   string
	Type      RXTX
	Signal    int
	Valid     bool
	Km        float64
	Mi        float64
	Location  string
}

const CHALLENGE_URL = "https://api.helium.io/v1/hotspots/%s/challenges"

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

// Download all the challenges from the API
func fetchChallenges(address string, count int) ([]Challenges, error) {
	challenges := make([]Challenges, count)
	totalChallenges := 0
	cursor := "" // keep track
	client := resty.New()

	for chalCount := 0; chalCount < count; {
		chals, c, err := getChallengeResponse(client, address, cursor)
		if err != nil {
			return []Challenges{}, fmt.Errorf("Unable to load challenges: %s", err)
		} else if totalChallenges == 0 && len(chals) == 0 && c == "" {
			// sometimes we get 0 results, but a cursor for "more"
			return []Challenges{}, fmt.Errorf("0 challenges fetched for %s.  Invalid address?", address)
		} else if len(chals) == 0 && c == "" {
			log.Warnf("Only able to retrieve %d challenges", totalChallenges)
			return challenges, nil
		}

		log.Debugf("Retrieved %d challenges", len(chals))
		chalCount += len(chals)
		cursor = c // keep track of the cursor for next time

		for i := 0; i < len(chals) && totalChallenges < count; i++ {
			challenges[totalChallenges] = chals[i]
			totalChallenges += 1
			if totalChallenges%100 == 0 {
				log.Infof("Loaded %d challenges", totalChallenges)
			}
		}
	}

	log.Debugf("found %d challenges for %s", len(challenges), address)
	return challenges, nil
}

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

func getWitnessResults(address, witness string, challenges []Challenges) ([]WitnessResult, error) {
	results := []WitnessResult{}
	for _, entry := range challenges {
		if entry.Type != "poc_receipts_v1" {
			log.Warnf("unexpected entry type: %s", entry.Type)
			continue
		}
		aHost, err := getHotspot(address)
		if err != nil {
			return []WitnessResult{}, err
		}

		for _, path := range *entry.Path {
			var rxtx RXTX = RX
			if path.Challengee != address {
				rxtx = TX
			}

			for _, wit := range *path.Witnesses {
				if wit.Gateway != witness {
					continue
				}

				wHost, err := getHotspot(witness)
				if err != nil {
					log.WithError(err).Errorf("Unable to lookup: %s", witness)
					continue
				}

				km, mi, err := getDistance(aHost, wHost)
				if err != nil {
					km = 0.0
					mi = 0.0
				}

				valid := true
				if float64(wit.Signal) > maxRssi(km) {
					valid = false
				}

				results = append(results, WitnessResult{
					Timestamp: int64(wit.Timestamp),
					Address:   address,
					Witness:   wit.Gateway,
					Signal:    wit.Signal,
					Type:      rxtx,
					Valid:     valid,
					Km:        km,
					Mi:        mi,
					Location:  wit.Location,
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
func writeChallenges(challenges []Challenges, filename, address string, cnt int) error {
	cache := ChallengeCache{
		Time:       time.Now().Unix(),
		Address:    address,
		Count:      cnt,
		Challenges: challenges,
	}
	b, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, b, 0644)
}

// read the cache file
func readChallenges(filename, address string, expires int64, cnt int) ([]Challenges, error) {
	cache := ChallengeCache{}
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return []Challenges{}, err
	}
	err = json.Unmarshal(bytes, &cache)
	if err != nil {
		return []Challenges{}, err
	}
	if cache.Time+expires < time.Now().Unix() {
		return []Challenges{}, fmt.Errorf("Challenge cache is old. Auto-refreshing...")
	}
	if cache.Count != cnt {
		return []Challenges{}, fmt.Errorf("Challenge cache stored %d instead of %d records.  Auto-refreshing...",
			cache.Count, cnt)
	}

	return cache.Challenges, nil
}

// returns the max RSSI based on distance
// Stolen from: https://github.com/Carniverous19/helium_analysis_tools.git
func maxRssi(km float64) float64 {
	if km < 0.001 {
		return -1000.0
	}
	return 28.0 + 1.8*2 - (20.0*math.Log10(km) + 20.0*math.Log10(915.0) + 32.44)
}

// Not sure why it is a list of values at the end???
// Table is map[SNR][0] = minimum valid RSSI
// Stolen from: https://github.com/Carniverous19/helium_analysis_tools.git
var SnrTable = map[int][]int{
	16:  {-90, -35},
	14:  {-90, -35},
	13:  {-90, -35},
	15:  {-90, -35},
	12:  {-90, -35},
	11:  {-90, -35},
	10:  {-90, -40},
	9:   {-95, -45},
	8:   {-105, -45},
	7:   {-108, -45},
	6:   {-113, -100},
	5:   {-115, -100},
	4:   {-115, -112},
	3:   {-115, -112},
	2:   {-117, -112},
	1:   {-120, -117},
	0:   {-125, -125},
	-1:  {-125, -125},
	-2:  {-125, -125},
	-3:  {-125, -125},
	-4:  {-125, -125},
	-5:  {-125, -125},
	-6:  {-124, -124},
	-7:  {-123, -123},
	-8:  {-125, -125},
	-9:  {-125, -125},
	-10: {-125, -125},
	-11: {-125, -125},
	-12: {-125, -125},
	-13: {-125, -125},
	-14: {-125, -125},
	-15: {-124, -124},
	-16: {-123, -123},
	-17: {-123, -123},
	-18: {-123, -123},
	-19: {-123, -123},
	-20: {-123, -123},
}

// returns the minimum valid RSSI at a given SNR
func minRssiPerSnr(snr float64) int {
	snri := int(math.Ceil(snr))
	v, ok := SnrTable[snri]
	if !ok {
		return 1000
	}
	return v[0]
}

// Get the haversine distance between two node addresses
func getDistance(aHost, bHost Hotspot) (float64, float64, error) {

	mi, km := haversine.Distance(
		haversine.Coord{
			Lat: aHost.Lat,
			Lon: aHost.Lng,
		},
		haversine.Coord{
			Lat: bHost.Lat,
			Lon: bHost.Lng,
		},
	)
	return km, mi, nil
}
