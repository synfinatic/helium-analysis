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
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/synfinatic/helium-analysis/analysis"
	"github.com/synfinatic/onelogin-aws-role/utils"
	bolt "go.etcd.io/bbolt"
)

type ChallengesCmd struct {
	Export    ChallengesExportCmd    `kong:"cmd,help='Export challenges from the database for given hotspot to JSON'"`
	Import    ChallengesImportCmd    `kong:"cmd,help='Import challenges into the database for given hotspot from JSON'"`
	Refresh   ChallengesRefreshCmd   `kong:"cmd,help='Refresh challenges in database for given hotspot'"`
	DeleteAll ChallengesDeleteAllCmd `kong:"cmd,help='Delete all challenges in database for given hotspot'"`
	Delete    ChallengesDeleteCmd    `kong:"cmd,help='Delete specified challenges in database for given hotspot'"`
	List      ChallengesListCmd      `kong:"cmd,help='List all the hotspots we have challenges for'"`
}

type ChallengesExportCmd struct {
	Address string `kong:"arg,required,help='Hotspot name or address to export'"`
	File    string `kong:"required,name='file',short='f',help='Output file for export'"`
}

type ChallengesImportCmd struct {
	Address string `kong:"arg,required,help='Hotspot name or address to import'"`
	File    string `kong:"required,name='file',short='f',help='Input JSON file to import'"`
}

type ChallengesRefreshCmd struct {
	Address string `kong:"arg,required,help='Hotspot name or address to refresh'"`
	Days    int64  `kong:"name='days',short='d',default=30,help='Previous number of days to load'"`
	Buffer  int64  `kong:"name='buffer',short='b',default=6,help='Challenge buffer in hours'"`
}

type ChallengesDeleteCmd struct {
	Address string `kong:"arg,required,help='Hotspot name or address to delete'"`
	Before  string `kong:"name='before',short='b',help='Delete data before YYYY-MM-DD',xor='mode'"`
	After   string `kong:"name='after',short='a',help='Delete data after YYYY-MM-DD',xor='mode'"`
}

type ChallengesDeleteAllCmd struct {
	Address string `kong:"arg,required,help='Hotspot name or address to delete'"`
}

type ChallengesListCmd struct {
	Local bool `kong:"name='localtime',default=false,help='Display in local time instead of UTC'"`
}

// Export challenges for a hostspot as JSON
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

// Sync latest challenges from api.helium.io
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

// Import challenges stored in JSON for a hotspot into the DB
func (cmd *ChallengesImportCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli

	address, err := ctx.BoltDB.GetHotspotByUnknown(cli.Challenges.Import.Address)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(cli.Challenges.Import.File)
	if err != nil {
		return err
	}

	challenges := []analysis.Challenges{}
	err = json.Unmarshal(data, &challenges)
	if err != nil {
		return err
	}

	db := ctx.BoltDB.GetDb()

	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := analysis.ChallengeBucket(tx, address)
		if err != nil {
			return err
		}

		for _, c := range challenges {
			jdata, err := json.Marshal(c)
			if err != nil {
				return err
			}
			key := make([]byte, 8)
			binary.BigEndian.PutUint64(key, uint64(c.Time))
			err = bucket.Put(key, jdata)
			if err != nil {
				return err
			}
		}

		return nil
	})
	return err
}

// Delete all challenges stored in the DB for a hotspot
func (cmd *ChallengesDeleteAllCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli

	address, err := ctx.BoltDB.GetHotspotByUnknown(cli.Challenges.DeleteAll.Address)
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

// returns true if key < t
func beforeKey(key []byte, t int64) bool {
	cmp := make([]byte, 8)
	binary.BigEndian.PutUint64(cmp, uint64(t))
	if bytes.Compare(key, cmp) < 0 {
		return true
	}
	return false
}

// Delete specified challenges stored in the DB for a hotspot
func (cmd *ChallengesDeleteCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli

	address, err := ctx.BoltDB.GetHotspotByUnknown(cli.Challenges.Delete.Address)
	if err != nil {
		return err
	}
	before := int64(0)
	after := int64(0)

	if cli.Challenges.Delete.Before != "" {
		t, err := time.Parse("2006-01-02", cli.Challenges.Delete.Before)
		if err != nil {
			return err
		}
		before = t.Unix()
	} else if cli.Challenges.Delete.After != "" {
		t, err := time.Parse("2006-01-02", cli.Challenges.Delete.After)
		if err != nil {
			return err
		}
		after = t.Unix()
	} else {
		return fmt.Errorf("Please specify --before or --after")
	}

	db := ctx.BoltDB.GetDb()
	err = db.Update(func(tx *bolt.Tx) error {
		baseBucket := tx.Bucket(analysis.CHALLENGES_BUCKET)
		bucket := baseBucket.Bucket([]byte(address))
		cursor := bucket.Cursor()

		if before > 0 {
			for key, _ := cursor.First(); beforeKey(key, before); key, _ = cursor.Next() {
				if key == nil {
					break
				}
				err := cursor.Delete()
				if err != nil {
					return err
				}
			}
		} else {
			first := make([]byte, 8)
			binary.BigEndian.PutUint64(first, uint64(after))
			for key, _ := cursor.Seek(first); key != nil; key, _ = cursor.Next() {
				err := cursor.Delete()
				if err != nil {
					return err
				}
			}

		}
		return nil
	})
	return err
}

// List all of the hotspots we have challenges for
func (cmd *ChallengesListCmd) Run(ctx *RunContext) error {
	cli := *ctx.Cli
	db := ctx.BoltDB.GetDb()
	report := []ChallengeReport{}

	err := db.View(func(tx *bolt.Tx) error {
		main := tx.Bucket(analysis.CHALLENGES_BUCKET)
		cursor := main.Cursor()
		for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
			address := string(k)

			// manually lookup the name
			bucket := tx.Bucket(analysis.HOTSPOTS_BUCKET)
			v := bucket.Get([]byte(address))
			if v == nil {
				return fmt.Errorf("%s is not in database", address)
			}
			hotspot := analysis.Hotspot{}
			err := json.Unmarshal(v, &hotspot)
			if err != nil {
				return err
			}
			name := hotspot.Name

			// find the first & last record
			challengeBucket := main.Bucket(k)
			c := challengeBucket.Cursor()
			first, _ := c.First()
			last, _ := c.Last()
			stats := challengeBucket.Stats()

			if first != nil {
				firstInt := int64(binary.BigEndian.Uint64(first))
				lastInt := int64(binary.BigEndian.Uint64(last))

				if cli.Challenges.List.Local {
					// Local time
					report = append(report, ChallengeReport{
						Name:    name,
						Address: address,
						First:   time.Unix(firstInt, 0).Format(analysis.TIME_FORMAT),
						Last:    time.Unix(lastInt, 0).Format(analysis.TIME_FORMAT),
						Records: int64(stats.KeyN),
					})
				} else {
					// UTC Time
					report = append(report, ChallengeReport{
						Name:    name,
						Address: address,
						First:   time.Unix(firstInt, 0).UTC().Format(analysis.TIME_FORMAT),
						Last:    time.Unix(lastInt, 0).UTC().Format(analysis.TIME_FORMAT),
						Records: int64(stats.KeyN),
					})
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// make a pretty table
	ts := []utils.TableStruct{}
	for _, r := range report {
		ts = append(ts, r)
	}
	fields := []string{
		"Name",
		"Address",
		"Records",
		"First",
		"Last",
	}
	utils.GenerateTable(ts, fields)
	fmt.Printf("\n")
	return err
}

// Necessary for utils.TableStruct magic
type ChallengeReport struct {
	Name    string `header:"Name"`
	Address string `header:"Address"`
	Records int64  `header:"Records"`
	First   string `header:"First"`
	Last    string `header:"Last"`
}

func (cr ChallengeReport) GetHeader(fieldName string) (string, error) {
	v := reflect.ValueOf(cr)
	return utils.GetHeaderTag(v, fieldName)
}
