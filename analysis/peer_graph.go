package analysis

/*
 * Helium Analysis
 * Copyright (c) 2021-2022 Aaron Turner  <aturner at synfin dot net>
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

// Generate all the peer graphs for a given address
func (b *BoltDB) GeneratePeerGraphs(address string, challenges []Challenges, settings GraphSettings) error {
	addresses, err := GetListOfAddresses(challenges)
	if err != nil {
		return err
	}

	if len(challenges) < settings.Min {
		return fmt.Errorf("Only %d datapoints available", len(challenges))
	}

	x_min := 0.0
	x_max := 0.0
	if !settings.Zoom {
		for i := 0; x_min == 0; i++ {
			min, err := challenges[i].GetTimestamp()
			if err == nil {
				x_min = float64(min)
			}
		}
		for i := len(challenges) - 1; x_max == 0; i-- {
			max, err := challenges[i].GetTimestamp()
			if err == nil {
				x_max = float64(max)
			}
		}
	}

	cnt := 0
	for _, peer := range addresses {
		wr, err := b.getWitnessResults(address, peer, challenges)
		if err != nil {
			log.WithError(err).Errorf("Unable to process: %s", peer)
			continue
		} else if len(wr) == 0 {
			// this tends to generate a LOT of messages since the list of challenges
			// has a lot of noise
			log.Debugf("Skipping %s <-> %s", address, peer)
			continue
		}

		var join_time int64 = 0
		host, err := b.GetHotspot(peer)
		if err == nil {
			join_time, err = getTimeForHeight(host.BlockAdded, challenges)
		}

		generated, err := b.generatePeerGraph(address, peer, wr, settings.Min, x_min, x_max, join_time, settings)
		if err != nil {
			log.WithError(err).Errorf("Unable to generate graph")
		}
		if generated {
			cnt += 1
		}
	}
	return nil
}

// Generate each peer graph
func (b *BoltDB) generatePeerGraph(address, witness string, results []WitnessResult, min int, x_min, x_max float64, join_time int64, settings GraphSettings) (bool, error) {
	a, err := b.GetHotspotName(address)
	if err != nil {
		return false, err
	}
	w, err := b.GetHotspotName(witness)
	if err != nil {
		return false, err
	}
	filename := fmt.Sprintf("%s/%s.png", a, w)
	jsonFilename := fmt.Sprintf("%s/%s.json", a, w)

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
	witnessName, err := b.GetHotspotName(witness)
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
		return false, nil
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

	wHotspot, _ := b.GetHotspot(witness)
	s := *wHotspot.Status
	status := fmt.Sprintf(" %s", s.Online)
	title := fmt.Sprintf("%s <=> %s (%.02fkm/%.02fmi) [%.02f]%s",
		a, w, results[0].Km, results[0].Mi, wHotspot.RewardScale, status)
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
		return false, fmt.Errorf("Unable to crate %s: %s", filename, err)
	}
	defer f.Close()
	graph.Render(chart.PNG, f)
	log.Infof("Created %s with %d data points", filename, dataPoints)

	if settings.Json {
		jdata, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return true, err
		}
		ioutil.WriteFile(jsonFilename, jdata, 0644)
	}
	return true, nil
}
