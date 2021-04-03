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
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/synfinatic/helium-analysis/analysis"
)

type ChallengesCmd struct {
	Action  string `kong:"name='action',enum='export',default='export',help='Available actions: export'"`
	Address string `kong:"name='address',short='a',help='Hotspot name or address to process'"`
	File    string `kong:"name='output',short='o',default='stdout',help='Output file for export'"`
}

func (cmd *ChallengesCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli

	// open our DB
	db, err := analysis.OpenDB(cli.Database)
	if err != nil {
		log.WithError(err).Fatalf("Unable to open database")
	}
	defer db.Close()

	// must call log.Panic() from now on!

	if cli.Challenges.Action == "export" {
		// Is this a name or address of a hotspot?  Set `hotspotAddress`
		hotspotAddress, err := db.GetHotspotByUnknown(cli.Challenges.Address)
		if err != nil {
			log.Panicf("%s", err)
		}

		challenges, err := db.GetChallenges(hotspotAddress, time.Unix(0, 0), time.Now())
		jdata, err := json.MarshalIndent(challenges, "", "  ")
		if err != nil {
			return err
		}

		if cli.Challenges.File == "stdout" {
			fmt.Printf("%s", string(jdata))
			return nil
		}
		return ioutil.WriteFile(cli.Challenges.File, jdata, 0644)
	}

	return fmt.Errorf("Unsupported action: %s", cli.Challenges.Action)
}
