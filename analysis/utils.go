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
	"math"
	"strconv"
	"time"

	"github.com/umahmood/haversine"
)

// used to merge the valid & invalid RX/TX data into a single series
func MergeTwoSeries(ax, ay, bx, by []float64) ([]float64, []float64) {
	alen := len(ax)
	blen := len(bx)
	newx := []float64{}
	newy := []float64{}

	j := 0
	for i := 0; i < alen; {
		// Must use 'a > b' because our data is in reverse!
		if j == blen || ax[i] > bx[j] {
			newx = append(newx, ax[i])
			newy = append(newy, ay[i])
			i += 1
		} else if j < blen {
			newx = append(newx, bx[j])
			newy = append(newy, by[j])
			j += 1
		}
	}
	for ; j < blen; j++ {
		newx = append(newx, bx[j])
		newy = append(newy, by[j])
	}
	return newx, newy
}

// Format time (X values)
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

func XValueFormatterUnix(v interface{}) string {
	if fv, isFloat := v.(float64); isFloat {
		t := time.Unix(int64(fv), 0)
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

// get a unique list of addresses in all the challenges
func GetListOfAddresses(challenges []Challenges) ([]string, error) {
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

// returns the max RSSI based on distance
// Stolen from: https://github.com/Carniverous19/helium_analysis_tools.git
func maxRssi(km float64) float64 {
	if km < 0.001 {
		return -1000.0
	}
	return 28.0 + 1.8*2 - (20.0 * math.Log10(km)) - (20.0 * math.Log10(915.0)) - 32.44
}

// Not sure why it is a list of values at the end???
// Table is map[SNR] = minimum valid RSSI
// Stolen from: https://github.com/Carniverous19/helium_analysis_tools.git
var SnrTable = map[int]int{
	16:  -90,
	14:  -90,
	13:  -90,
	15:  -90,
	12:  -90,
	11:  -90,
	10:  -90,
	9:   -95,
	8:   -105,
	7:   -108,
	6:   -113,
	5:   -115,
	4:   -115,
	3:   -115,
	2:   -117,
	1:   -120,
	0:   -125,
	-1:  -125,
	-2:  -125,
	-3:  -125,
	-4:  -125,
	-5:  -125,
	-6:  -124,
	-7:  -123,
	-8:  -125,
	-9:  -125,
	-10: -125,
	-11: -125,
	-12: -125,
	-13: -125,
	-14: -125,
	-15: -124,
	-16: -123,
	-17: -123,
	-18: -123,
	-19: -123,
	-20: -123,
}

// returns the minimum valid RSSI at a given SNR
func minRssiPerSnr(snr float64) float64 {
	snri := int(math.Ceil(snr))
	v, ok := SnrTable[snri]
	if !ok {
		return 1000.0
	}
	return float64(v)
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
