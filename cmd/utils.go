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
	"strconv"
	"time"
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
