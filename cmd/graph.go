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
	"fmt"
	"os"
	"strconv"
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
	HEIGHT   = 512
	WIDTH    = 1024
	Y_MIN    = -130.0
	Y_MAX    = -70.0
	DOT_SIZE = 5
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

func generatePeerGraph(address, witness string, results []WitnessResult, min int, x_min, x_max float64) (error, bool) {
	a, err := getHotspotName(address)
	if err != nil {
		return err, false
	}
	b, err := getHotspotName(witness)
	if err != nil {
		return err, false
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
		DotWidth:    DOT_SIZE,
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
		return nil, false
	}

	// Zoom in?
	x_range := chart.ContinuousRange{}
	y_range := chart.ContinuousRange{}
	if x_min > 0.0 && x_max > 0.0 {
		x_range.Min = x_min
		x_range.Max = x_max
		y_range.Min = Y_MIN
		y_range.Max = Y_MAX
	}

	title := fmt.Sprintf("%s <=> %s (%.02fkm/%.02fmi)", a, b, results[0].Km, results[0].Mi)
	graph := chart.Chart{
		Title: title,
		Background: chart.Style{
			Padding: chart.Box{
				Top:    110,
				Left:   20,
				Right:  20,
				Bottom: 10,
			},
		},
		XAxis: chart.XAxis{
			ValueFormatter: XValueFormatter,
			Range:          &x_range,
		},
		YAxis: chart.YAxis{
			Range: &y_range,
		},
		Height: HEIGHT,
		Width:  WIDTH,
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.LegendThin(&graph),
	}
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Unable to crate %s: %s", filename, err), false
	}
	defer f.Close()
	graph.Render(chart.PNG, f)
	log.Infof("Created %s with %d data points", filename, dataPoints)
	return nil, true
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

func generatePeerGraphs(address string, challenges []Challenges, min int, zoom bool) {
	addresses, err := getListOfAddresses(challenges)
	if err != nil {
		log.WithError(err).Fatalf("Unable to get addresses")
	}

	x_min := 0.0
	x_max := 0.0
	if !zoom {
		c := *challenges[0].Path
		p := *c[0].Witnesses
		x_max = float64(p[0].Timestamp)
		last := len(challenges) - 1
		c = *challenges[last].Path
		p = *c[0].Witnesses
		x_min = float64(p[0].Timestamp)
	}

	cnt := 0
	for _, peer := range addresses {
		wr, err := getWitnessResults(address, peer, challenges)
		if err != nil {
			log.WithError(err).Errorf("Unable to process: %s", peer)
			continue
		}

		err, generated := generatePeerGraph(address, peer, wr, min, x_min, x_max)
		if err != nil {
			log.WithError(err).Errorf("Unable to generate graph")
		}
		if generated {
			cnt += 1
		}
	}
}

func XValueFormatter(v interface{}) string {
	if fv, isFloat := v.(float64); isFloat {
		t := time.Unix(int64(fv/1000000000), 0)
		hr, _ := strconv.Atoi(t.Format("15"))
		htime := "am"
		if hr > 11 {
			htime = "pm"
			if hr > 12 {
				hr -= 12
			}
		}
		return fmt.Sprintf("%s %d%s", t.Format("2006-01-02"), hr, htime)
	}
	return ""
}
