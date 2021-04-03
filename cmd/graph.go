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
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/synfinatic/helium-analysis/analysis"
)

type GraphCmd struct {
	Address string `kong:"arg,required,name='address',help='Hotspot address or name to report on'"`
	Days    int64  `kong:"name='days',default=30,help='Previous number of days to report on'"`
	Last    string `kong:"name='last',default='1h',help='Age of last challenge before looking for more challenges'"`
	Minimum int    `kong:"name='minimum',default=5,help='Minimum required challenges to generate a graph'"`
	Json    bool   `kong:"name='json',default=false,help='Generate per-hotspot JSON files'"`
}

func (cmd *GraphCmd) Run(ctx *RunContext) error {
	var hotspotAddress string
	cli := *ctx.Cli

	// validate --minimum
	if cli.Graph.Minimum < 2 {
		log.Fatalf("Please specify a --minimum value >= 2")
	}

	// validate --days and set `firstTime`
	if cli.Graph.Days < 1 {
		log.Fatalf("Please specify a --days value >= 1")
	}
	daysOffset := time.Now().Unix() - (cli.Graph.Days * int64(24*60*60))
	days := time.Unix(daysOffset, 0)
	// go to the beginning of the day UTC
	startDate := days.Format("2006-01-02")
	firstTime, _ := time.Parse("2006-01-02", startDate)

	// validate --last and set `lastTime`
	lastTime, err := parseLastTime(cli.Graph.Last)
	if err != nil {
		log.Fatalf("%s", err)
	}

	// open our DB
	db, err := analysis.OpenDB(cli.Database)
	if err != nil {
		log.WithError(err).Fatalf("Unable to open database")
	}
	defer db.Close()

	// must call log.Panic() from now on!

	// Is this a name or address of a hotspot?  Set `hotspotAddress`
	var a, b, c string
	n, err := fmt.Scanf("%s-%s-%s", cli.Graph.Address, a, b, c)
	if n == 3 && err == nil {
		// user provided hotspot name
		hotspotAddress, err = db.GetHotspotAddress(cli.Graph.Address)
		if err != nil {
			log.WithError(err).Panicf("Invalid hotspot name '%s'.  Refresh hotspot cache?", cli.Graph.Address)
		}
	} else {
		_, err = db.GetHotspotName(cli.Graph.Address)
		if err != nil {
			log.WithError(err).Panicf("Invalid hotspot address '%s'  Refresh hotspot cache?", cli.Graph.Address)
		}
		hotspotAddress = cli.Graph.Address
	}

	err = db.LoadChallenges(hotspotAddress, firstTime, lastTime)
	if err != nil {
		log.WithError(err).Warnf("Unable to refresh challenges.  Using cache.")
	}

	_, err = db.GetChallenges(hotspotAddress, firstTime, lastTime)
	if err != nil {
		log.WithError(err).Panic("Unable to load challenges")
	}

	return nil
}

// Calc time offset based on refresh string
func parseLastTime(last string) (time.Time, error) {
	var x int
	var t string
	n, err := fmt.Scanf("%d%s", last, x, t)
	if err != nil {
		return time.Now(), fmt.Errorf("Unable to parse --last %s: %s", last, err)
	}
	if n != 2 {
		return time.Now(), fmt.Errorf("Invalid --last %s.  Must be integer followed by: d, h, m", last)
	}

	if x < 0 {
		return time.Now(), fmt.Errorf("Invalid --last %s.  Must be positive integer value", last)
	}

	// convert into seconds
	switch t {
	case "m":
		x *= 60
	case "h":
		x *= 60 * 60
	case "d":
		x *= 60 * 60 * 24
	}

	// calc time offset
	offsetSecs := time.Now().Unix() - int64(x)
	return time.Unix(offsetSecs, 0), nil
}
