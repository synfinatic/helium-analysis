package main

import (
	"fmt"
	"os"
	"time"

	"github.com/wcharczuk/go-chart/v2"

	log "github.com/sirupsen/logrus"
)

type RXTX int

const (
	RX RXTX = iota
	TX
)

const (
	//	HEIGHT = 1535
	//	WIDTH  = 2048
	HEIGHT = 512
	WIDTH  = 1024
)

// Creates the PNG for the given args
func generateGraph(address string, direction RXTX, results []ChallengeResult, filename string) {
	series := []chart.Series{}
	var label string

	addresses, err := getAddresses(results)
	if err != nil {
		log.WithError(err).Fatalf("Unable to get addresses")
	}
	for _, addr := range addresses {
		if direction == RX {
			label = "RX"
		} else {
			label = "TX"
		}

		x, y := getSignalsTimeSeriesChallenge(addr, results)
		h, err := getHotspot(addr)
		name := addr
		if err != nil {
			log.WithError(err).Warnf("Unable to get Hotspot info")
		} else {
			name = h.Name
		}

		series = append(series, chart.TimeSeries{
			Name:    fmt.Sprintf("%s %s", name, label),
			XValues: x,
			YValues: y,
		})
	}

	graph := chart.Chart{
		Background: chart.Style{
			Padding: chart.Box{
				Top:  20,
				Left: 260,
			},
		},
		Height: HEIGHT,
		Width:  WIDTH,
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.LegendLeft(&graph),
	}
	f, err := os.Create(filename)
	if err != nil {
		log.WithError(err).Fatalf("Unable to crate %s", filename)
	}
	defer f.Close()
	graph.Render(chart.PNG, f)
	log.Infof("Created %s", filename)
}

func generatePeerGraph(address, witness string, results []WitnessResult) error {
	a, err := getHotspotName(address)
	if err != nil {
		return err
	}
	b, err := getHotspotName(witness)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s:%s.png", a, b)

	tx_x := []time.Time{}
	rx_x := []time.Time{}

	tx_vals := []float64{}
	rx_vals := []float64{}
	tx_valid_vals := []float64{}
	rx_valid_vals := []float64{}

	for _, ret := range results {
		var valid float64 = 1.0
		if !ret.Valid {
			valid = 0.0
		}

		if ret.Type == RX {
			rx_x = append(rx_x, time.Unix(ret.Timestamp/1000000000, 0))
			rx_vals = append(rx_vals, float64(ret.Signal))
			rx_valid_vals = append(rx_valid_vals, valid)
		} else {
			tx_x = append(tx_x, time.Unix(ret.Timestamp/1000000000, 0))
			tx_vals = append(tx_vals, float64(ret.Signal))
			tx_valid_vals = append(tx_valid_vals, valid)
		}
	}

	dataPoints := len(tx_x) + len(rx_x)
	if dataPoints < 2 {
		// no data
		log.Debugf("Skipping: %s", filename)
		return nil
	}

	graph := chart.Chart{
		Background: chart.Style{
			Padding: chart.Box{
				Top:  20,
				Left: 20,
			},
		},
		Height: HEIGHT,
		Width:  WIDTH,
		Series: []chart.Series{
			chart.TimeSeries{
				Name:    "TX Signal",
				XValues: tx_x,
				YValues: tx_vals,
			},
			chart.TimeSeries{
				Name:    "RX Signal",
				XValues: rx_x,
				YValues: rx_vals,
			},
			/*
				chart.TimeSeries{
					Name:    "RX Valid",
					XValues: rx_x,
					YValues: rx_valid_vals,
				},
				chart.TimeSeries{
					Name:    "TX Valid",
					XValues: tx_x,
					YValues: tx_valid_vals,
				},
			*/
		},
	}

	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Unable to crate %s: %s", filename, err)
	}
	defer f.Close()
	graph.Render(chart.PNG, f)
	log.Infof("Created %s with %d data points", filename, dataPoints)
	return nil
}

func getListOfAddresses(challenges []Challenges) ([]string, error) {
	addrs := map[string]int{}
	for _, chal := range challenges {
		p := *chal.Path
		for _, witness := range *p[0].Witnesses {
			_, ok := addrs[witness.Gateway]
			if !ok {
				addrs[witness.Gateway] = 1
			} else {
				addrs[witness.Gateway] += 1
			}

		}
	}
	ret := []string{}
	for k, _ := range addrs {
		ret = append(ret, k)
	}
	return ret, nil
}

func generatePeerGraphs(address string, challenges []Challenges) {
	addresses, err := getListOfAddresses(challenges)
	if err != nil {
		log.WithError(err).Fatalf("Unable to get addresses")
	}
	for _, peer := range addresses {
		wr, err := getWitnessResults(address, peer, challenges)
		if err != nil {
			log.WithError(err).Errorf("Unable to process: %s", peer)
			continue
		}

		err = generatePeerGraph(address, peer, wr)
		if err != nil {
			log.WithError(err).Errorf("Unable to generate graph")
		}
	}
}
