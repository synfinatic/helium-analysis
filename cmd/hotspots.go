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

	log "github.com/sirupsen/logrus"
	"github.com/synfinatic/helium-analysis/analysis"
)

type HotspotsCmd struct {
	Action string `kong:"name='action',enum='refresh,export',default='export',help='Available actions: refresh,export'"`
	File   string `kong:"name='output',short='o',default='stdout',help='Output file for export'"`
}

func (cmd *HotspotsCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli

	// open our DB
	db, err := analysis.OpenDB(cli.Database)
	if err != nil {
		log.WithError(err).Fatalf("Unable to open database")
	}
	defer db.Close()

	// must call log.Panic() from now on!

	if cli.Hotspots.Action == "refresh" {
		hotspots, err := analysis.FetchHotspots()
		if err != nil {
			log.WithError(err).Panicf("Unable to fetch hotspots")
		}

		return db.SetAllHotspots(hotspots)
	}

	if cli.Hotspots.Action == "export" {
		hotspots, err := db.GetHotspots()
		if err != nil {
			return fmt.Errorf("Unable to get hotspots: %s", err)
		}

		jdata, err := json.MarshalIndent(hotspots, "", "  ")
		if err != nil {
			return err
		}

		if cli.Hotspots.File == "stdout" {
			fmt.Printf("%s", string(jdata))
			return nil
		} else {
			return ioutil.WriteFile(cli.Hotspots.File, jdata, 0644)
		}

	}

	return fmt.Errorf("Unknown command: %s", cli.Hotspots.Action)
}
