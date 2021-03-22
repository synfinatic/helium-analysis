package main

import (
	//	"github.com/wcharczuk/go-chart/v2" // exposes chart
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

func main() {
	var debug bool
	var version bool
	var address string
	var challenges int

	flag.BoolVar(&debug, "debug", false, "Enable debugging")
	flag.BoolVar(&version, "version", false, "Print version and exit")
	flag.StringVar(&address, "address", "", "Hotspot address to report (required)")
	flag.IntVar(&challenges, "challenges", 500, "Number of challenges to proecess")

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
		fmt.Printf("Helium Analysis Version %s -- Copyright 2021 Aaron Turner\n", Version)
		fmt.Printf("%s (%s) built at %s\n", CommitID, Tag, Buildinfos)
		os.Exit(0)
	}

	if address == "" {
		log.Fatalf("Please specify --address")
	}

	c := getChallenges(address, challenges)
	writeChallenges(c, "challenges.json")
}
