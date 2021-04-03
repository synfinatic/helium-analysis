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

	"github.com/alecthomas/kong"
	"github.com/mattn/go-colorable"
	log "github.com/sirupsen/logrus"
)

var Version = "unknown"
var Buildinfos = "unknown"
var Tag = "NO-TAG"
var CommitID = "unknown"

const (
	CHALLENGES_CACHE_EXPIRES = 1 // 1 hr
	HOTSPOT_CACHE_FILE       = "hotspots.json"
	DATABASE_FILE            = "helium.db"
)

type RunContext struct {
	Ctx *kong.Context
	Cli *CLI
}

type CLI struct {
	// Common Arguments
	LogLevel string `kong:"optional,short='l',name='loglevel',default='info',enum='error,warn,info,debug',help='Logging level [error|warn|info|debug]'"`
	Database string `kong:"optional,short='d',name='database',default='helium.db',help='Database file'"`

	// sub commands
	Graph      GraphCmd      `kong:"cmd,help='Generate graphs for the given hotspot'"`
	Hotspots   HotspotsCmd   `kong:"cmd,help='Manage hotspots in database'"`
	Challenges ChallengesCmd `kong:"cmd,help='Manage challenges in database'"`
	Version    VersionCmd    `kong:"cmd,help='Print version and exit'"`
}

func main() {
	op := kong.Description("Helium Analysis")
	cli := CLI{}
	ctx := kong.Parse(&cli, op)

	switch cli.LogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
	case "info":
		log.SetLevel(log.InfoLevel)
		log.SetOutput(colorable.NewColorableStdout())
	case "warn":
		log.SetLevel(log.WarnLevel)
		log.SetOutput(colorable.NewColorableStdout())
	case "error":
		log.SetLevel(log.ErrorLevel)
		log.SetOutput(colorable.NewColorableStdout())
	}

	run_ctx := RunContext{
		Ctx: ctx,
		Cli: &cli,
	}
	err := ctx.Run(&run_ctx)
	if err != nil {
		log.Fatalf("Error running command: %s", err.Error())
	}
}

// Version Command
type VersionCmd struct{}

func (cmd *VersionCmd) Run(ctx *RunContext) error {
	fmt.Printf("Helium Analysis v%s -- Copyright 2021 Aaron Turner\n", Version)
	fmt.Printf("%s (%s) built at %s\n", CommitID, Tag, Buildinfos)
	fmt.Printf("\nIf you find this useful, please donate a few HNT to:\n")
	fmt.Printf("144xaKFbp4arCNWztcDbB8DgWJFCZxc8AtAKuZHZ6Ejew44wL8z")
	return nil
}
