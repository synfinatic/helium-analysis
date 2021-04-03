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
	//	"fmt"
	//	"time"

	log "github.com/sirupsen/logrus"
	"github.com/synfinatic/helium-analysis/analysis"
)

type HotspotsCmd struct{}

func (cmd *HotspotsCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli

	// open our DB
	db, err := analysis.OpenDB(cli.Database)
	if err != nil {
		log.WithError(err).Fatalf("Unable to open database")
	}
	defer db.Close()

	// must call log.Panic() from now on!

	hotspots, err := analysis.FetchHotspots()
	if err != nil {
		log.WithError(err).Panicf("Unable to fetch hotspots")
	}

	return db.SetAllHotspots(hotspots)
}
