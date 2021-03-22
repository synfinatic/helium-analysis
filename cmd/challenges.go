package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

type ChallengeResponse struct {
	Data   []Challenges `json:data`
	Cursor string       `json:cursor`
}

type Challenges struct {
	Type               string  `json:type`
	Time               int     `json:time`
	Secret             string  `json:secret`
	Path               []Path  `json:path`
	OnionKeyHash       string  `json:onion_key_hash`
	Height             int     `json:height`
	Hash               string  `json:hash`
	Fee                int     `json:fee`
	ChallengerOwner    string  `json:challenger_owner`
	ChallengerLon      float64 `json:challenger_lon`
	ChallengerLat      float64 `json:challenger_lat`
	ChallengerLocation string  `json:challenger_location`
	Challenger         string  `json:challenger`
}

type Path struct {
	Witnesses          []WitnessType `json:witnesses`
	Receipt            ReceiptType   `json:receipt`
	Geocode            GeocodeType   `json:geocode`
	ChallengeeOwner    string        `json:challengee_owner`
	ChallengeeLon      float64       `json:challengee_lon`
	ChallengeeLat      float64       `json:challengee_lat`
	ChallengeeLocation string        `json:challengee_location`
	Challengee         string        `json:challengee`
}

type WitnessType struct {
	Timestamp  int64  `json:timestamp`
	Signal     int    `json:signal`
	PacketHash string `json:packet_hash`
	Owner      string `json:owner`
	Location   string `json:location`
	Gateway    string `json:gateway`
}

type ReceiptType struct {
	Timestamp int64  `json:timestamp`
	Signal    int    `json:signal`
	Origin    string `json:origin`
	Gateway   string `json:gateway`
	Data      string `json:data`
}

type GeocodeType struct {
	ShortStreet  string `json:short_street`
	ShortState   string `json:short_state`
	ShortCountry string `json:short_country`
	ShortCity    string `json:short_city`
	Street       string `json:long_street`
	State        string `json:long_state`
	Country      string `json:long_country`
	City         string `json:long_city`
}

const API_URL = "https://api.helium.io/v1/hotspots/%s/challenges"

// Returns a list of Challenges and the cursor location or an error
func getResponse(client *resty.Client, address string, cursor string) ([]Challenges, string, error) {
	var resp *resty.Response
	var err error

	if cursor == "" {
		resp, err = client.R().
			SetHeader("Accept", "application/json").
			SetResult(&ChallengeResponse{}).
			Get(fmt.Sprintf(API_URL, address))
	} else {
		resp, err = client.R().
			SetHeader("Accept", "application/json").
			SetResult(&ChallengeResponse{}).
			SetQueryParams(map[string]string{
				"cursor": cursor,
			}).
			Get(fmt.Sprintf(API_URL, address))
	}
	if err != nil {
		return []Challenges{}, "", err
	}
	result := (resp.Result().(*ChallengeResponse))

	return result.Data, result.Cursor, nil
}

// Fails to return on error
func getChallenges(address string, count int) []Challenges {
	challenges := make([]Challenges, count)
	totalChallenges := 0
	cursor := "" // keep track
	client := resty.New()

	for chalCount := 0; chalCount < count; {
		chals, c, err := getResponse(client, address, cursor)
		if err != nil {
			log.WithError(err).Fatalf("Unable to load challenges")
		}
		log.Debugf("Retrieved %d challenges", len(chals))
		chalCount += len(chals)
		cursor = c // keep track of the cursor for next time

		for i := 0; i < len(chals) && totalChallenges < count; i++ {
			challenges[totalChallenges] = chals[i]
			totalChallenges += 1
			if totalChallenges%100 == 0 {
				log.Info("Loaded %d challenges", totalChallenges)
			}
		}
	}

	return challenges
}

// generates a ton of output
func printChallenges(challenges []Challenges) {
	fmt.Sprintf(spew.Sdump(challenges))
}

// use this as a cache for later?
func writeChallenges(challenges []Challenges, filename string) error {
	b, err := json.MarshalIndent(challenges, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, b, 0644)
}
