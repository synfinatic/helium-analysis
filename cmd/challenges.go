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
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"time"
)

type ChallengesExportCmd struct {
	Address string `kong:"arg,required,help='Hotspot name or address to process'"`
	File    string `kong:"name='output',short='o',default='stdout',help='Output file for export'"`
}

type ChallengesRefreshCmd struct {
	Address string `kong:"arg,required,help='Hotspot name or address to process'"`
	Days    int64  `kong:"name='days',short='d',default=30,help='Previous number of days to load'"`
}

type ChallengesCmd struct {
	Export  ChallengesExportCmd  `kong:"cmd,help='Export challenges for given hotspot as JSON'"`
	Refresh ChallengesRefreshCmd `kong:"cmd,help='Refresh challenges in database for given hotspot'"`
}

func (cmd *ChallengesExportCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli

	// Is this a name or address of a hotspot?  Set `hotspotAddress`
	hotspotAddress, err := ctx.BoltDB.GetHotspotByUnknown(cli.Challenges.Export.Address)
	if err != nil {
		return err
	}

	challenges, err := ctx.BoltDB.GetChallenges(hotspotAddress, time.Unix(0, 0), time.Now())
	if err != nil {
		return err
	}

	log.Infof("Retrieved %d challenges", len(challenges))
	if len(challenges) == 0 {
		return nil
	}

	jdata, err := json.MarshalIndent(challenges, "", "  ")
	if err != nil {
		return err
	}

	if cli.Challenges.Export.File != "stdout" {
		return ioutil.WriteFile(cli.Challenges.Export.File, jdata, 0644)
	}
	fmt.Printf("%s", string(jdata))
	return nil
}

func (cmd *ChallengesRefreshCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli

	// Is this a name or address of a hotspot?  Set `hotspotAddress`
	hotspotAddress, err := ctx.BoltDB.GetHotspotByUnknown(cli.Challenges.Refresh.Address)
	if err != nil {
		return err
	}

	daysOffset := time.Now().Unix() - (cli.Challenges.Refresh.Days * int64(24*60*60))
	days := time.Unix(daysOffset, 0)
	// go to the beginning of the day UTC
	startDate := days.Format("2006-01-02")
	firstTime, _ := time.Parse("2006-01-02", startDate)
	lastTime := time.Now()

	return ctx.BoltDB.LoadChallenges(hotspotAddress, firstTime, lastTime)
}
