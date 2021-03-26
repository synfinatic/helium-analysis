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

		x, y := getSignalsSeriesChallenge(addr, results)
		h, err := getHotspot(addr)
		name := addr
		if err != nil {
			log.WithError(err).Warnf("Unable to get Hotspot info")
		} else {
			name = h.Name
		}

		series = append(series, chart.ContinuousSeries{
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

func generatePeerGraph(address, witness string, results []WitnessResult, min int) error {
	a, err := getHotspotName(address)
	if err != nil {
		return err
	}
	b, err := getHotspotName(witness)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s:%s.png", a, b)

	tx_x := []float64{}
	rx_x := []float64{}

	tx_vals := []float64{}
	rx_vals := []float64{}
	tx_valid_vals := []chart.Value2{}
	rx_valid_vals := []chart.Value2{}

	for _, ret := range results {
		x := float64(ret.Timestamp)
		y := float64(ret.Signal)
		if ret.Type == RX {
			rx_x = append(rx_x, x)
			rx_vals = append(rx_vals, y)
			if !ret.Valid {
				rx_valid_vals = append(rx_valid_vals, chart.Value2{XValue: x, YValue: y, Label: "Invalid"})
			}
		} else {
			tx_x = append(tx_x, x)
			tx_vals = append(tx_vals, y)
			if !ret.Valid {
				tx_valid_vals = append(tx_valid_vals, chart.Value2{XValue: x, YValue: y, Label: "Invalid"})
			}
		}
	}

	series := []chart.Series{}
	style := chart.Style{
		StrokeWidth: chart.Disabled,
		DotWidth:    5,
	}
	dataPoints := 0
	if len(tx_x) >= min {
		txSeries := chart.ContinuousSeries{
			Name:    "TX Signal",
			Style:   style,
			XValues: tx_x,
			YValues: tx_vals,
		}

		txSmaSeries := &chart.SMASeries{
			Name:        "TX Average",
			InnerSeries: txSeries,
		}

		txValidSeries := chart.AnnotationSeries{
			Annotations: tx_valid_vals,
		}
		series = append(series, txSeries, txSmaSeries, txValidSeries)
		dataPoints += len(tx_x)
	}

	if len(rx_x) >= min {
		rxSeries := chart.ContinuousSeries{
			Name:    "RX Signal",
			Style:   style,
			XValues: rx_x,
			YValues: rx_vals,
		}

		rxSmaSeries := &chart.SMASeries{
			Name:        "RX Average",
			InnerSeries: rxSeries,
		}

		rxValidSeries := chart.AnnotationSeries{
			Annotations: rx_valid_vals,
		}
		series = append(series, rxSeries, rxSmaSeries, rxValidSeries)
		dataPoints += len(rx_x)
	}

	if len(series) == 0 {
		// no data
		log.Debugf("Skipping: %s", filename)
		return nil
	}

	title := fmt.Sprintf("%s <=> %s (%.02fkm/%.02fmi)", a, b, results[0].Km, results[0].Mi)
	graph := chart.Chart{
		Title: title,
		Background: chart.Style{
			Padding: chart.Box{
				Top:  70,
				Left: 20,
			},
		},
		XAxis: chart.XAxis{
			ValueFormatter: func(v interface{}) string {
				if fv, isFloat := v.(float64); isFloat {
					t := time.Unix(int64(fv/1000000000), 0)
					return t.Format("2006-01-02 15:04")
				}
				return ""
			},
		},
		Height: HEIGHT,
		Width:  WIDTH,
		Series: series,
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

func generatePeerGraphs(address string, challenges []Challenges, min int) {
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

		err = generatePeerGraph(address, peer, wr, min)
		if err != nil {
			log.WithError(err).Errorf("Unable to generate graph")
		}
	}
}
