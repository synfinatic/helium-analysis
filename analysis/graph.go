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
	"fmt"
	"os"

	"github.com/wcharczuk/go-chart/v2"

	log "github.com/sirupsen/logrus"
)

type RXTX int

type GraphSettings struct {
	Min  int  // minimum challenges
	Zoom bool // zoom in
	Json bool // generate json for each pair
}

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
func (b *BoltDB) GenerateBeaconsGraph(address string, results []Challenges, settings GraphSettings) error {
	hotspotName, err := b.GetHotspotName(address)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s/beacon-totals.png", hotspotName)

	x_data := []float64{}
	valid_data := []float64{}
	invalid_data := []float64{}
	for _, challenge := range results {
		path := *challenge.Path
		if path[0].Challengee != address { // find only our beacons
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
		if valid == 0 && invalid == 0 {
			continue
		}
		valid_data = append(valid_data, float64(valid))
		invalid_data = append(invalid_data, float64(invalid))
		x_data = append(x_data, float64(challenge.Time))
	}

	if len(x_data) < settings.Min {
		return fmt.Errorf("Only %d datapoints available", len(x_data))
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
		Min: x_data[0],
		Max: x_data[len(x_data)-1],
	}
	graph := chart.Chart{
		Title:  fmt.Sprintf("Beacon Totals for %s", hotspotName),
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
		YAxis: chart.YAxis{
			Name: "total witnesses",
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
func (b *BoltDB) GenerateWitnessesGraph(address string, results []Challenges, settings GraphSettings) error {
	hotspotName, err := b.GetHotspotName(address)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s/witness-distance.png", hotspotName)
	host, err := b.GetHotspot(address)
	if err != nil {
		return err
	}

	x_valid := []float64{}
	x_invalid := []float64{}
	valid_data := []float64{}
	invalid_data := []float64{}
	for _, challenge := range results {
		path := *challenge.Path
		if path[0].Challengee == address {
			continue // ignore where we are the one sending the beacon
		}
		for _, witness := range *path[0].Witnesses {
			if witness.Gateway == address {
				otherHost, err := b.GetHotspot(path[0].Challengee)
				if err != nil {
					log.Errorf("Unable to find %s", path[0].Challengee)
					continue
				}
				km, _, _ := getDistance(host, otherHost)
				if witness.IsValid {
					valid_data = append(valid_data, float64(km))
					x_valid = append(x_valid, float64(challenge.Time))
					break
				} else {
					invalid_data = append(invalid_data, float64(km))
					x_invalid = append(x_invalid, float64(challenge.Time))
					break
				}
			}
		}
	}

	if (len(x_valid) + len(x_invalid)) < settings.Min {
		return fmt.Errorf("Only %d datapoints available", len(x_valid)+len(x_invalid))
	}

	validSeries := chart.ContinuousSeries{
		Name: "Valid",
		Style: chart.Style{
			StrokeWidth: chart.Disabled,
			DotWidth:    2,
			StrokeColor: chart.ColorGreen,
			DotColor:    chart.ColorGreen,
		},
		XValues: x_valid,
		YValues: valid_data,
	}

	invalidSeries := chart.ContinuousSeries{
		Name: "Invalid",
		Style: chart.Style{
			StrokeWidth: chart.Disabled,
			DotWidth:    2,
			DotColor:    chart.ColorRed,
			StrokeColor: chart.ColorRed,
		},
		XValues: x_invalid,
		YValues: invalid_data,
	}

	series := []chart.Series{
		validSeries,
		invalidSeries,
	}
	graph := chart.Chart{
		Title:  fmt.Sprintf("Witness Result for %s", hotspotName),
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
		},
		YAxis: chart.YAxis{
			Name: "km",
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
