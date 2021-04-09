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

type HotspotsExportCmd struct {
	File string `kong:"name='output',short='o',default='stdout',help='Output file for export'"`
}

type HotspotsRefreshCmd struct{}

type HotspotsCmd struct {
	Export  HotspotsExportCmd  `kong:"cmd,help='Export hotspots as JSON'"`
	Refresh HotspotsRefreshCmd `kong:"cmd,help='Refresh hotspots database cache'"`
}

func (cmd *HotspotsExportCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli
	hotspots, err := ctx.BoltDB.GetHotspots()
	if err != nil {
		return fmt.Errorf("Unable to get hotspots: %s", err)
	}

	jdata, err := json.MarshalIndent(hotspots, "", "  ")
	if err != nil {
		return err
	}

	if cli.Hotspots.Export.File != "stdout" {
		return ioutil.WriteFile(cli.Hotspots.Export.File, jdata, 0644)
	}

	fmt.Printf("%s", string(jdata))
	return nil
}

func (cmd *HotspotsRefreshCmd) Run(ctx *RunContext) error {
	//	cli := *ctx.Cli

	hotspots, err := analysis.FetchHotspots()
	if err != nil {
		log.WithError(err).Panicf("Unable to fetch hotspots")
	}

	return ctx.BoltDB.SetAllHotspots(hotspots)
}
