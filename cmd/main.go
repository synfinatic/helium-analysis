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

	"github.com/mattn/go-colorable"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var Version = "unknown"
var Buildinfos = "unknown"
var Tag = "NO-TAG"
var CommitID = "unknown"

const CHALLENGES_CACHE_EXPIRES = 1 // 1 hr

func main() {
	var debug, version, zoom, noCache, forceCache, generateJson bool
	var address, challengeCache, name string
	var min, days int
	var challengesExpires int64
	var err error

	flag.BoolVar(&debug, "debug", false, "Enable debugging")
	flag.BoolVar(&version, "version", false, "Print version and exit")
	flag.StringVar(&address, "address", "", "Hotspot address to report on")
	flag.StringVar(&name, "name", "", "Hotspot name to report on")
	flag.IntVar(&days, "days", 30, "Set starting point in days")
	flag.IntVar(&min, "min", 5, "Minimum challenges required to graph")
	//	flag.BoolVar(&zoom, "zoom", false, "Unfix X/Y axis to zoom in")
	flag.StringVar(&challengeCache, "cache", "", "Challenges cache file")
	flag.Int64Var(&challengesExpires, "expires", CHALLENGES_CACHE_EXPIRES, "Challenge cache timeout (hrs)")
	flag.BoolVar(&noCache, "no-cache", false, "Disable loading/reading challenges cache")
	flag.BoolVar(&forceCache, "force-cache", false, "Force using existing cache and skip API calls")
	flag.BoolVar(&generateJson, "json", false, "Generate per-graph JSON reports")
	flag.Parse()

	if debug == true {
		log.SetReportCaller(true)
		log.SetLevel(log.DebugLevel)
	} else {
		// pretty console output
		log.SetLevel(log.InfoLevel)
		log.SetFormatter(&log.TextFormatter{ForceColors: true})
		log.SetOutput(colorable.NewColorableStdout())
	}

	if version == true {
		fmt.Printf("Helium Analysis v%s -- Copyright 2021 Aaron Turner\n", Version)
		fmt.Printf("%s (%s) built at %s\n", CommitID, Tag, Buildinfos)
		fmt.Printf("\nIf you find this useful, please donate a few HNT to:\n")
		fmt.Printf("144xaKFbp4arCNWztcDbB8DgWJFCZxc8AtAKuZHZ6Ejew44wL8z")
		os.Exit(0)
	}

	if min < 2 {
		log.Fatalf("Please specify a --min value >= 2")
	}

	refreshHotspots := false
	err, tooOld := loadHotspots(HOTSPOT_CACHE_FILE)
	if err != nil {
		log.WithError(err).Warn("Unable to load hotspot cache.  Refreshing...")
		refreshHotspots = true
	} else if tooOld && !forceCache {
		log.Infof("Refreshing...")
		refreshHotspots = true
	}

	if refreshHotspots {
		err = downloadHotspots(HOTSPOT_CACHE_FILE)
		if err != nil {
			log.WithError(err).Fatalf("Unable to load hotspots.")
		}
		err, _ = loadHotspots(HOTSPOT_CACHE_FILE)
		if err != nil {
			log.WithError(err).Fatalf("Unable to load new hotspot cache")
		}
	}

	if name == "" && address == "" {
		log.Fatalf("Please specify --address or --name")
	}

	if name != "" {
		address, err = getHotspotAddress(name)
		if err != nil {
			log.Fatalf("%s", err)
		}
	} else {
		name, err = getHotspotName(address)
		if err != nil {
			log.Fatalf("%s", err)
		}
	}

	// set cache file if not set
	if challengeCache == "" {
		challengeCache = fmt.Sprintf("%s.json", name)
	}

	// Calcuate the beginning of the day, X days back
	hours, _ := time.ParseDuration(fmt.Sprintf("%dh", days*24))
	startTime := time.Now().Add(-hours)
	startDate := startTime.Format("2006-01-02")
	start, _ := time.Parse("2006-01-02", startDate)

	c := []Challenges{}
	if noCache {
		c, err = fetchChallenges(address, start)
		if err != nil {
			log.Fatalf("%s", err)
		}
	} else {
		c, err = loadChallenges(challengeCache, address, challengesExpires*3600, start, forceCache)
		if err != nil {
			log.WithError(err).Warnf("Unable to load challenges file. Refreshing...")
			c, err = fetchChallenges(address, start)
			if err != nil {
				log.Fatalf("%s", err)
			}
		}
		if !noCache {
			writeChallenges(c, challengeCache, address, start)
		}
	}

	generatePeerGraphs(address, c, min, zoom, generateJson)
	err = generateBeaconsGraph(address, c)
	if err != nil {
		log.WithError(err).Errorf("Unable to generate beacons graph")
	}
	err = generateWitnessesGraph(address, c)
	if err != nil {
		log.WithError(err).Errorf("Unable to generate witnesses graph")
	}
}
