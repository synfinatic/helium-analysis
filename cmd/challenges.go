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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/synfinatic/helium-analysis/analysis"
	bolt "go.etcd.io/bbolt"
)

type ChallengesExportCmd struct {
	Address string `kong:"arg,required,help='Hotspot name or address to export'"`
	File    string `kong:"name='output',short='o',default='stdout',help='Output file for export'"`
}

type ChallengesRefreshCmd struct {
	Address string `kong:"arg,required,help='Hotspot name or address to refresh'"`
	Days    int64  `kong:"name='days',short='d',default=30,help='Previous number of days to load'"`
	Buffer  int64  `kong:"name='buffer',short='b',default=6,help='Challenge buffer in hours'"`
}

type ChallengesDeleteCmd struct {
	Address string `kong:"arg,required,help='Hotspot name or address to delete'"`
}

type ChallengesListCmd struct {
	Local bool `kong:"name='localtime',default=false,help='Display in local time instead of UTC'"`
}

type ChallengesCmd struct {
	Export  ChallengesExportCmd  `kong:"cmd,help='Export challenges for given hotspot as JSON'"`
	Refresh ChallengesRefreshCmd `kong:"cmd,help='Refresh challenges in database for given hotspot'"`
	Delete  ChallengesDeleteCmd  `kong:"cmd,help='Delete all challenges in database for given hotspot'"`
	List    ChallengesListCmd    `kong:"cmd,help='List all the hotspots we have challenges for'"`
}

func (cmd *ChallengesExportCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli

	// Is this a name or address of a hotspot?  Set `hotspotAddress`
	hotspotAddress, err := ctx.BoltDB.GetHotspotByUnknown(cli.Challenges.Export.Address)
	if err != nil {
		return err
	}

	challenges, err := ctx.BoltDB.GetChallenges(hotspotAddress, time.Unix(0, 0), time.Now().UTC())
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

	daysOffset := time.Now().UTC().Unix() - (cli.Challenges.Refresh.Days * int64(24*60*60))
	days := time.Unix(daysOffset, 0)
	// go to the beginning of the day UTC
	startDate := days.Format("2006-01-02")
	firstTime, _ := time.Parse("2006-01-02", startDate)
	lastTime := time.Now().UTC()
	duration := time.Duration(time.Hour * time.Duration(cli.Challenges.Refresh.Buffer))

	return ctx.BoltDB.LoadChallenges(hotspotAddress, firstTime, lastTime, duration)
}

func (cmd *ChallengesDeleteCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli

	address, err := ctx.BoltDB.GetHotspotByUnknown(cli.Challenges.Delete.Address)
	if err != nil {
		return err
	}

	db := ctx.BoltDB.GetDb()
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(analysis.CHALLENGES_BUCKET)
		err := bucket.DeleteBucket([]byte(address))
		if err != nil {
			log.WithError(err).Warnf("Unable to delete bucket")
		}
		return nil
	})
	return err
}

func (cmd *ChallengesListCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli
	db := ctx.BoltDB.GetDb()
	buckets := map[string][]int64{}

	err := db.View(func(tx *bolt.Tx) error {
		main := tx.Bucket(analysis.CHALLENGES_BUCKET)
		cursor := main.Cursor()
		for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
			name := string(k)
			challengeBucket := main.Bucket(k)
			c := challengeBucket.Cursor()
			first, _ := c.First()
			last, _ := c.Last()
			if first != nil {
				buckets[name] = append(buckets[name], int64(binary.BigEndian.Uint64(first)))
				buckets[name] = append(buckets[name], int64(binary.BigEndian.Uint64(last)))
			} else {
				buckets[name] = append(buckets[name], 0)
				buckets[name] = append(buckets[name], 0)
			}
		}

		return nil
	})

	for k, v := range buckets {
		name, err := ctx.BoltDB.GetHotspotName(k)
		if err != nil {
			name = "Unknown"
		}
		if v[0] != v[1] {
			var first, last string
			if cli.Challenges.List.Local {
				first = time.Unix(v[0], 0).Format(analysis.UTC_FORMAT)
				last = time.Unix(v[1], 0).Format(analysis.UTC_FORMAT)
			} else {
				first = time.Unix(v[0], 0).UTC().Format(analysis.UTC_FORMAT)
				last = time.Unix(v[1], 0).UTC().Format(analysis.UTC_FORMAT)
			}
			fmt.Printf("%s %s\t%s => %s\n", name, k, first, last)
		} else {
			fmt.Printf("%s %s\tNo records", name, k)
		}
	}
	return err
}
