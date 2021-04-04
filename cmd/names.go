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
	bolt "go.etcd.io/bbolt"
)

type NamesExportCmd struct {
	File string `kong:"name='output',short='o',default='stdout',help='Output file for export'"`
}

type NamesCmd struct {
	Export NamesExportCmd `kong:"cmd,help='Export hotspots as JSON'"`
}

func (cmd *NamesExportCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli

	// open our DB
	db, err := analysis.OpenDB(cli.Database)
	if err != nil {
		log.WithError(err).Fatalf("Unable to open database")
	}
	defer db.Close()

	// must call log.Panic() from now on!

	names := map[string]string{}

	b := db.GetDb()
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(analysis.HOTSPOT_NAMES_BUCKET)
		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			names[string(k)] = string(v)
		}
		return nil
	})

	if err != nil {
		return err
	}

	jdata, err := json.MarshalIndent(names, "", "  ")
	if err != nil {
		return err
	}

	if cli.Names.Export.File != "stdout" {
		return ioutil.WriteFile(cli.Names.Export.File, jdata, 0644)
	}

	fmt.Printf("%s", string(jdata))
	return nil
}
