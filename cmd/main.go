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

func main() {
	var debug, version, hotspots, challengeCache, zoom bool
	var address string
	var min, challengesCnt int

	flag.BoolVar(&debug, "debug", false, "Enable debugging")
	flag.BoolVar(&version, "version", false, "Print version and exit")
	flag.StringVar(&address, "address", "", "Hotspot address to report (required)")
	flag.IntVar(&challengesCnt, "challenges", 500, "Number of challenges to process")
	flag.BoolVar(&hotspots, "hotspots", false, "Download a current list of hotspots and exit")
	flag.IntVar(&min, "min", 5, "Minimum challenges required to graph")
	flag.BoolVar(&challengeCache, "use-cache", false, fmt.Sprintf("Use %s cache file", CHALLENGES_CACHE_FILE))
	flag.BoolVar(&zoom, "zoom", false, "Unfix X/Y axis to zoom in")

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
	if challengeCache {
		c, err = readChallenges(CHALLENGES_CACHE_FILE)
		if err != nil {
			log.WithError(err).Fatalf("Unable to load challenges file")
		}
	} else {
		c = getChallenges(address, challengesCnt)
		writeChallenges(c, CHALLENGES_CACHE_FILE)
	}

	generatePeerGraphs(address, c, min, zoom)
}
