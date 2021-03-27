package main

import (
	"fmt"
	"os"

	"github.com/mattn/go-colorable"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var Version = "unknown"
var Buildinfos = "unknown"
var Tag = "NO-TAG"
var CommitID = "unknown"

const CHALLENGES_CACHE_FILE = "challenges.json"
const CHALLENGES_CACHE_EXPIRES = 1 // 1 hr

func main() {
	var debug, version, hotspots, zoom, noCache bool
	var address, challengeCache string
	var min, challengesCnt int
	var challengesExpires int64

	flag.BoolVar(&debug, "debug", false, "Enable debugging")
	flag.BoolVar(&version, "version", false, "Print version and exit")
	flag.StringVar(&address, "address", "", "Hotspot address to report (required)")
	flag.IntVar(&challengesCnt, "challenges", 500, "Number of challenges to process")
	flag.BoolVar(&hotspots, "hotspots", false, "Download a current list of hotspots and exit")
	flag.IntVar(&min, "min", 5, "Minimum challenges required to graph")
	flag.BoolVar(&zoom, "zoom", false, "Unfix X/Y axis to zoom in")
	flag.StringVar(&challengeCache, "cache", CHALLENGES_CACHE_FILE, "Challenges cache file")
	flag.Int64Var(&challengesExpires, "expires", CHALLENGES_CACHE_EXPIRES, "Challenge cache timeout (hrs)")
	flag.BoolVar(&noCache, "no-cache", false, "Disable loading/reading challenges cache")

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
		os.Exit(0)
	}

	if hotspots {
		err := downloadHotspots(HOTSPOT_CACHE_FILE)
		if err != nil {
			log.WithError(err).Fatalf("Unable to load hotspots")
		}
		os.Exit(0)
	}

	if address == "" {
		log.Fatalf("Please specify --address")
	}

	err := loadHotspots(HOTSPOT_CACHE_FILE)
	if err != nil {
		log.WithError(err).Fatalf("Unable to load hotspots")
	}

	c := []Challenges{}
	if noCache {
		c, err = fetchChallenges(address, challengesCnt)
		if err != nil {
			log.Fatalf("%s", err)
		}
	} else {
		c, err = readChallenges(CHALLENGES_CACHE_FILE, address, challengesExpires*3600, challengesCnt)
		if err != nil {
			log.WithError(err).Warnf("Unable to load challenges file. Falling back to download.")
			c, err = fetchChallenges(address, challengesCnt)
			if err != nil {
				log.Fatalf("%s", err)
			}
		}
		if !noCache {
			writeChallenges(c, CHALLENGES_CACHE_FILE, address, challengesCnt)
		}

	}

	generatePeerGraphs(address, c, min, zoom)
}
