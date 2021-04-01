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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

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
	DOT_SIZE = 3
	SNR_MIN  = -20.0
	SNR_MAX  = 16.0
)

// Creates the PNG for the the beacons sent
func generateBeaconsGraph(address string, results []Challenges) error {
	hotspotName, err := getHotspotName(address)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s:beacons.png", hotspotName)

	x_data := []float64{}
	valid_data := []float64{}
	invalid_data := []float64{}
	for _, challenge := range results {
		path := *challenge.Path
		if path[0].Challengee != address {
			continue
		}
		valid := 0
		invalid := 0
		for _, witness := range *path[0].Witnesses {
			if witness.Gateway == address {
				continue
			}
			if witness.IsValid {
				valid += 1
			} else {
				invalid += 1
			}
		}
		valid_data = append(valid_data, float64(valid))
		invalid_data = append(invalid_data, float64(invalid))
		x_data = append(x_data, float64(challenge.Time))
	}

	validSeries := chart.ContinuousSeries{
		Name: "Valid",
		Style: chart.Style{
			StrokeColor: chart.ColorGreen,
		},
		XValues: x_data,
		YValues: valid_data,
	}

	validSma := chart.SMASeries{
		Style: chart.Style{
			StrokeWidth:     1,
			DotWidth:        chart.Disabled,
			StrokeColor:     chart.ColorGreen,
			StrokeDashArray: []float64{5.0, 5.0},
		},
		InnerSeries: validSeries,
	}

	invalidSeries := chart.ContinuousSeries{
		Name: "Invalid",
		Style: chart.Style{
			StrokeColor: chart.ColorRed,
		},
		XValues: x_data,
		YValues: invalid_data,
	}

	invalidSma := chart.SMASeries{
		Style: chart.Style{
			StrokeWidth:     1,
			DotWidth:        chart.Disabled,
			StrokeColor:     chart.ColorRed,
			StrokeDashArray: []float64{5.0, 5.0},
		},
		InnerSeries: invalidSeries,
	}

	series := []chart.Series{
		validSeries,
		validSma,
		invalidSeries,
		invalidSma,
	}
	x_range := chart.ContinuousRange{
		Max: x_data[0],
		Min: x_data[len(x_data)-1],
	}
	graph := chart.Chart{
		Title:  fmt.Sprintf("Beacons for %s", hotspotName),
		Height: HEIGHT,
		Width:  WIDTH,
		Series: series,
		Background: chart.Style{
			Padding: chart.Box{
				Top:    110,
				Left:   20,
				Right:  20,
				Bottom: 10,
			},
		},
		XAxis: chart.XAxis{
			ValueFormatter: XValueFormatterUnix,
			Range:          &x_range,
		},
	}

	graph.Elements = []chart.Renderable{
		chart.LegendThin(&graph),
	}
	f, err := os.Create(filename)
	if err != nil {
		log.WithError(err).Fatalf("Unable to crate %s", filename)
	}
	defer f.Close()
	graph.Render(chart.PNG, f)
	log.Infof("Created %s", filename)
	return nil
}

// Creates the PNG for the the witnesses
func generateWitnessesGraph(address string, results []Challenges) error {
	hotspotName, err := getHotspotName(address)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s:witnesses.png", hotspotName)

	x_data := []float64{}
	valid_data := []float64{}
	invalid_data := []float64{}
	for _, challenge := range results {
		path := *challenge.Path
		if path[0].Challengee == address {
			continue
		}
		valid := 0
		invalid := 0
		for _, witness := range *path[0].Witnesses {
			if witness.Gateway == address {
				continue
			}
			if witness.IsValid {
				valid += 1
			} else {
				invalid += 1
			}
		}
		valid_data = append(valid_data, float64(valid))
		invalid_data = append(invalid_data, float64(invalid))
		x_data = append(x_data, float64(challenge.Time))
	}

	validSeries := chart.ContinuousSeries{
		Name: "Valid",
		Style: chart.Style{
			StrokeWidth: chart.Disabled,
			DotWidth:    chart.Disabled,
		},
		XValues: x_data,
		YValues: valid_data,
	}

	validSma := chart.SMASeries{
		Style: chart.Style{
			StrokeColor: chart.ColorGreen,
		},
		InnerSeries: validSeries,
	}

	invalidSeries := chart.ContinuousSeries{
		Style: chart.Style{
			StrokeWidth: chart.Disabled,
			DotWidth:    chart.Disabled,
		},
		XValues: x_data,
		YValues: invalid_data,
	}

	invalidSma := chart.SMASeries{
		Name:        "Invalid",
		InnerSeries: invalidSeries,
		Style: chart.Style{
			StrokeWidth: 1,
			DotWidth:    chart.Disabled,
			StrokeColor: chart.ColorRed,
		},
	}

	series := []chart.Series{
		validSeries,
		validSma,
		invalidSeries,
		invalidSma,
	}
	x_range := chart.ContinuousRange{
		Max: x_data[0],
		Min: x_data[len(x_data)-1],
	}
	graph := chart.Chart{
		Title:  fmt.Sprintf("Witnesses Avg for %s", hotspotName),
		Height: HEIGHT,
		Width:  WIDTH,
		Series: series,
		Background: chart.Style{
			Padding: chart.Box{
				Top:    110,
				Left:   20,
				Right:  20,
				Bottom: 10,
			},
		},
		XAxis: chart.XAxis{
			ValueFormatter: XValueFormatterUnix,
			Range:          &x_range,
		},
	}

	graph.Elements = []chart.Renderable{
		chart.LegendThin(&graph),
	}
	f, err := os.Create(filename)
	if err != nil {
		log.WithError(err).Fatalf("Unable to crate %s", filename)
	}
	defer f.Close()
	graph.Render(chart.PNG, f)
	log.Infof("Created %s", filename)
	return nil
}

func generatePeerGraph(address, witness string, results []WitnessResult, min int, x_min, x_max float64, join_time int64, generateJson bool) (error, bool) {
	a, err := getHotspotName(address)
	if err != nil {
		return err, false
	}
	b, err := getHotspotName(witness)
	if err != nil {
		return err, false
	}
	filename := fmt.Sprintf("%s:%s.png", a, b)
	jsonFilename := fmt.Sprintf("%s:%s.json", a, b)

	thresholds_x := []float64{}
	thresholds_vals := []float64{}
	snr := []float64{}

	tx_x := []float64{}
	tx_vals := []float64{}
	rx_x := []float64{}
	rx_vals := []float64{}
	tx_invalid_x := []float64{}
	tx_invalid_vals := []float64{}
	rx_invalid_x := []float64{}
	rx_invalid_vals := []float64{}

	forceYRange := 1000.0
	forceSNRRange := 1000.0
	witnessName, err := getHotspotName(witness)
	if err != nil {
		witnessName = witness
	}
	for _, ret := range results {
		x := float64(ret.Timestamp)
		y := float64(ret.Signal)
		if y < Y_MIN || y > Y_MAX {
			forceYRange = y
		}
		if ret.Type == RX {
			if !ret.Valid {
				rx_invalid_x = append(rx_invalid_x, x)
				rx_invalid_vals = append(rx_invalid_vals, y)
			} else {
				rx_x = append(rx_x, x)
				rx_vals = append(rx_vals, y)
			}
		} else {
			if !ret.Valid {
				tx_invalid_x = append(tx_invalid_x, x)
				tx_invalid_vals = append(tx_invalid_vals, y)
			} else {
				tx_x = append(tx_x, x)
				tx_vals = append(tx_vals, y)
			}
		}
		thresholds_vals = append(thresholds_vals, ret.ValidThreshold)
		thresholds_x = append(thresholds_x, x)
		snr = append(snr, ret.Snr)
		if ret.Snr < SNR_MIN || ret.Snr > SNR_MAX {
			forceSNRRange = ret.Snr
		}
	}

	series := []chart.Series{}

	dataPoints := 0
	if len(tx_x) >= min {
		txSeries := chart.ContinuousSeries{
			Name: fmt.Sprintf("TX Signal (%d)", len(tx_x)),
			Style: chart.Style{
				StrokeWidth: chart.Disabled,
				DotWidth:    DOT_SIZE,
				StrokeColor: chart.ColorBlue,
				DotColor:    chart.ColorBlue,
			},
			XValues: tx_x,
			YValues: tx_vals,
		}

		hidden_x, hidden_y := MergeTwoSeries(tx_x, tx_vals, tx_invalid_x, tx_invalid_vals)
		txHiddenSeries := chart.ContinuousSeries{
			XValues: hidden_x,
			YValues: hidden_y,
			Style: chart.Style{
				Hidden: true,
			},
		}

		txSmaSeries := &chart.SMASeries{
			Style: chart.Style{
				StrokeWidth:     1,
				DotWidth:        chart.Disabled,
				StrokeColor:     chart.ColorBlue,
				DotColor:        chart.ColorBlue,
				StrokeDashArray: []float64{5.0, 5.0},
			},
			InnerSeries: txHiddenSeries,
		}

		txInvalidSeries := chart.ContinuousSeries{
			Name: fmt.Sprintf("InvalidTX (%d)", len(tx_invalid_x)),
			Style: chart.Style{
				StrokeWidth: chart.Disabled,
				DotWidth:    DOT_SIZE,
				DotColor:    chart.ColorRed,
				StrokeColor: chart.ColorRed,
			},
			XValues: tx_invalid_x,
			YValues: tx_invalid_vals,
		}
		series = append(series, txSeries, txSmaSeries, txInvalidSeries)
		dataPoints += len(tx_x) + len(tx_invalid_x)
	}

	if len(rx_x) >= min {
		rxSeries := chart.ContinuousSeries{
			Name: fmt.Sprintf("RX Signal (%d)", len(rx_x)),
			Style: chart.Style{
				StrokeWidth: chart.Disabled,
				DotWidth:    DOT_SIZE,
				StrokeColor: chart.ColorGreen,
				DotColor:    chart.ColorGreen,
			},
			XValues: rx_x,
			YValues: rx_vals,
		}

		hidden_x, hidden_y := MergeTwoSeries(rx_x, rx_vals, rx_invalid_x, rx_invalid_vals)
		rxHiddenSeries := chart.ContinuousSeries{
			XValues: hidden_x,
			YValues: hidden_y,
			Style: chart.Style{
				Hidden: true,
			},
		}

		rxSmaSeries := &chart.SMASeries{
			Style: chart.Style{
				StrokeWidth:     1,
				DotWidth:        chart.Disabled,
				StrokeColor:     chart.ColorGreen,
				DotColor:        chart.ColorGreen,
				StrokeDashArray: []float64{5.0, 5.0},
			},
			InnerSeries: rxHiddenSeries,
		}

		rxInvalidSeries := chart.ContinuousSeries{
			Name: fmt.Sprintf("InvalidRX (%d)", len(rx_invalid_x)),
			Style: chart.Style{
				StrokeWidth: chart.Disabled,
				DotWidth:    DOT_SIZE,
				DotColor:    chart.ColorYellow,
				StrokeColor: chart.ColorYellow,
			},
			XValues: rx_invalid_x,
			YValues: rx_invalid_vals,
		}
		series = append(series, rxSeries, rxSmaSeries, rxInvalidSeries)
		dataPoints += len(rx_x) + len(rx_invalid_x)
	}

	if len(series) == 0 {
		// no data
		log.Debugf("Skipping: %s", filename)
		return nil, false
	}

	if err != nil {
		log.Fatalf("%s", err)
	}
	thresholdSeries := chart.ContinuousSeries{
		Name:    "MinValid RSSI",
		XValues: thresholds_x,
		YValues: thresholds_vals,
		Style: chart.Style{
			StrokeWidth:     1,
			DotWidth:        chart.Disabled,
			StrokeColor:     chart.ColorOrange,
			StrokeDashArray: []float64{5.0, 5.0},
		},
	}
	series = append(series, thresholdSeries)
	series = append(series, chart.ContinuousSeries{
		Name:    fmt.Sprintf("MaxValid RSSI (%.02f)", maxRssi(results[0].Km)),
		XValues: []float64{thresholds_x[0], thresholds_x[len(thresholds_x)-1]},
		YValues: []float64{maxRssi(results[0].Km), maxRssi(results[0].Km)},
		Style: chart.Style{
			StrokeWidth:     1,
			DotWidth:        chart.Disabled,
			StrokeColor:     chart.ColorRed,
			StrokeDashArray: []float64{5.0, 5.0},
		},
	})

	snrSeries := chart.ContinuousSeries{
		YAxis:   chart.YAxisSecondary,
		Name:    "SNR",
		XValues: thresholds_x,
		YValues: snr,
		Style: chart.Style{
			StrokeWidth:     1,
			DotWidth:        chart.Disabled,
			StrokeColor:     chart.ColorYellow,
			StrokeDashArray: []float64{5.0, 5.0},
		},
	}
	series = append(series, snrSeries)

	lockYRange := true
	if forceYRange != 1000.0 {
		log.Warnf("% 2.0fdB is outside of graph defaults.  Unlocking primary Y axis for %s",
			forceYRange, witnessName)
		lockYRange = false
	}
	lockSNRRange := true
	if forceSNRRange != 1000.0 {
		log.Warnf("% 2.0fdB is outside of graph defaults.  Unlocking secondary (SNR) Y axis for %s",
			forceSNRRange, witnessName)
		lockSNRRange = false
	}

	// Zoom in?
	x_range := chart.ContinuousRange{}
	y_range := chart.ContinuousRange{}
	snr_range := chart.ContinuousRange{}
	if x_min > 0.0 && x_max > 0.0 {
		x_range.Min = x_min
		x_range.Max = x_max
		if lockYRange {
			y_range.Min = Y_MIN
			y_range.Max = Y_MAX
		}
		if lockSNRRange {
			snr_range.Min = SNR_MIN
			snr_range.Max = SNR_MAX
		}
	}

	// marker for when hotspot joined
	if join_time > 0 && float64(join_time) > x_min {
		series = append(series,
			chart.AnnotationSeries{
				Annotations: []chart.Value2{
					{XValue: float64(join_time), YValue: -70.0, Label: "Joined"},
				},
			},
		)
	}

	w, _ := getHotspot(witness)
	s := *w.Status
	status := fmt.Sprintf(" %s", s.Online)
	title := fmt.Sprintf("%s <=> %s (%.02fkm/%.02fmi) [%.02f]%s", a, b, results[0].Km, results[0].Mi, w.RewardScale, status)
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
		YAxisSecondary: chart.YAxis{
			Name:  "SNR db",
			Range: &snr_range,
		},
		YAxis: chart.YAxis{
			Name:  "RSSI db",
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

	if generateJson {
		jdata, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err, true
		}
		ioutil.WriteFile(jsonFilename, jdata, 0644)
	}
	return nil, true
}

func generatePeerGraphs(address string, challenges []Challenges, min int, zoom, generateJson bool) {
	addresses, err := GetListOfAddresses(challenges)
	if err != nil {
		log.WithError(err).Fatalf("Unable to get addresses")
	}

	x_min := 0.0
	x_max := 0.0
	if !zoom {
		for i := 0; x_max == 0; i++ {
			max, err := challenges[i].GetTimestamp()
			if err == nil {
				x_max = float64(max)
			}
		}
		for i := len(challenges) - 1; x_min == 0; i-- {
			min, err := challenges[i].GetTimestamp()
			if err == nil {
				x_min = float64(min)
			}
		}
	}

	cnt := 0
	for _, peer := range addresses {
		wr, err := getWitnessResults(address, peer, challenges)
		if err != nil {
			log.WithError(err).Errorf("Unable to process: %s", peer)
			continue
		}

		var join_time int64 = 0
		host, err := getHotspot(peer)
		if err == nil {
			join_time, err = getTimeForHeight(host.BlockAdded, challenges)
		}

		err, generated := generatePeerGraph(address, peer, wr, min, x_min, x_max, join_time, generateJson)
		if err != nil {
			log.WithError(err).Errorf("Unable to generate graph")
		}
		if generated {
			cnt += 1
		}
	}
}
