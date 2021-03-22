package main

import (
	"fmt"
	"os"

	"github.com/wcharczuk/go-chart/v2"

	log "github.com/sirupsen/logrus"
)

type RXTX int

const (
	RX RXTX = iota
	TX
)

// Creates the PNG for the given args
func generateGraph(address string, direction RXTX, results []ChallengeResults, filename string) {
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

		x, y := getSignalsTimeSeries(addr, results)
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
}
