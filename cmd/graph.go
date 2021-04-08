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
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/synfinatic/helium-analysis/analysis"
)

type GraphCmd struct {
	Address     string `kong:"arg,required,name='address',help='Hotspot address or name to report on'"`
	Days        int64  `kong:"name='days',short='d',default=30,help='Previous number of days to report on'"`
	Last        string `kong:"name='last',short='l',default='1h',help='Age of last challenge before looking for more challenges'"`
	Minimum     int    `kong:"name='minimum',short='m',default=5,help='Minimum required challenges to generate a graph'"`
	Json        bool   `kong:"name='json',short='j',default=false,help='Generate per-hotspot JSON files'"`
	Buffer      int64  `kong:"name='buffer',short='b',default=6,help='Challenge buffer in hours'"`
	SkipRefresh bool   `kong:"name='skip-refresh',default=false,help='Skip refresh of hotspot data via api.helium.io'"`
}

func (cmd *GraphCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli

	// validate --minimum
	if cli.Graph.Minimum < 2 {
		return fmt.Errorf("Please specify a --minimum value >= 2")
	}

	// validate --days and set `firstTime`
	if cli.Graph.Days < 1 {
		return fmt.Errorf("Please specify a --days value >= 1")
	}
	daysOffset := time.Now().UTC().Unix() - (cli.Graph.Days * int64(24*60*60))
	days := time.Unix(daysOffset, 0).UTC()
	// go to the beginning of the day UTC
	startDate := days.Format("2006-01-02")
	firstTime, _ := time.Parse("2006-01-02", startDate)

	// validate --last and set `lastTime`
	lastTime, err := parseLastTime(cli.Graph.Last)
	if err != nil {
		return err
	}
	log.Debugf("start: %s\t\tend: %s", firstTime.Format(analysis.UTC_FORMAT),
		lastTime.Format(analysis.UTC_FORMAT))

	// Is this a name or address of a hotspot?  Set `hotspotAddress`
	hotspotAddress, err := ctx.BoltDB.GetHotspotByUnknown(cli.Graph.Address)
	if err != nil {
		return err
	}

	name, err := ctx.BoltDB.GetHotspotName(hotspotAddress)
	if err != nil {
		return err
	}

	err = makeDirectory(name)
	if err != nil {
		return err
	}

	if !cli.Graph.SkipRefresh {
		duration := time.Duration(time.Hour * time.Duration(cli.Graph.Buffer))
		err = ctx.BoltDB.LoadChallenges(hotspotAddress, firstTime, lastTime, duration)
		if err != nil {
			log.WithError(err).Warnf("Unable to refresh challenges.  Using cache.")
		}
	}

	challenges, err := ctx.BoltDB.GetChallenges(hotspotAddress, firstTime, lastTime)
	if err != nil {
		log.WithError(err).Panic("Unable to load challenges")
	}

	settings := analysis.GraphSettings{
		Min:  cli.Graph.Minimum,
		Zoom: false,
		Json: cli.Graph.Json,
	}

	err = ctx.BoltDB.GenerateBeaconsGraph(hotspotAddress, challenges, settings)
	if err != nil {
		log.WithError(err).Error("Unable to generate beacons graph")
	}

	err = ctx.BoltDB.GenerateWitnessesGraph(hotspotAddress, challenges, settings)
	if err != nil {
		log.WithError(err).Error("Unable to generate witnesses graph")
	}

	err = ctx.BoltDB.GeneratePeerGraphs(hotspotAddress, challenges, settings)
	if err != nil {
		log.WithError(err).Error("Unable to generate peer graph(s)")
	}
	return nil
}

// Calc time offset based on refresh string
func parseLastTime(last string) (time.Time, error) {
	var x int
	var t string

	n, err := fmt.Sscanf(last, "%d%s", &x, &t)
	if err != nil {
		return time.Now().UTC(), fmt.Errorf("Unable to parse --last %s: %s", last, err)
	}
	if n != 2 {
		return time.Now().UTC(), fmt.Errorf("Invalid --last %s.  Must be integer followed by: d, h, m", last)
	}

	if x < 0 {
		return time.Now().UTC(), fmt.Errorf("Invalid --last %s.  Must be positive integer value", last)
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
	offsetSecs := time.Now().UTC().Unix() - int64(x)
	return time.Unix(offsetSecs, 0), nil
}

// Make the directory for the given address
func makeDirectory(name string) error {
	var err error = nil
	stat, err := os.Stat(name)
	if os.IsNotExist(err) {
		err = os.Mkdir(name, 0755)
	} else if !stat.IsDir() {
		err = fmt.Errorf("%s already exists and is not a directory", name)
	}
	return err
}
