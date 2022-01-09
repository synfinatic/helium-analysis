package main

/*
 * Helium Analysis
 * Copyright (c) 2021-2022 Aaron Turner  <aturner at synfin dot net>
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

	//	log "github.com/sirupsen/logrus"
	"github.com/synfinatic/helium-analysis/analysis"
	bolt "go.etcd.io/bbolt"
)

type NamesExportCmd struct {
	File string `kong:"name='output',short='o',default='stdout',help='Output file for export'"`
}

type NamesAddressCmd struct {
	Name string `kong:"arg,required,help='Name of hotspot'"`
}

type NamesCmd struct {
	Export  NamesExportCmd  `kong:"cmd,help='Export hotspots as JSON'"`
	Address NamesAddressCmd `kong:"cmd,help='Lookup address for hotspot name'"`
}

// Export all the names => address as a json map
func (cmd *NamesExportCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli
	names := map[string]string{}

	b := ctx.BoltDB.GetDb()
	err := b.View(func(tx *bolt.Tx) error {
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

// print a single name => address
func (cmd *NamesAddressCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli

	address, err := ctx.BoltDB.GetHotspotAddress(cli.Names.Address.Name)
	if err != nil {
		return err
	}
	fmt.Printf("%s => %s\n", cli.Names.Address.Name, string(address))

	return err
}
