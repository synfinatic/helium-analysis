# Helium Analysis

Some simple tools for optmizing your [Helium](https://www.helium.com) hotspot
antenna and location by exmaining your PoC signals (beacons & witnesses).

## What?

Creates pretty graphs of PoC hotspot activity.   Here are a few examples:

Beacon total witness count over time:
![](https://user-images.githubusercontent.com/1075352/113659393-c6b47080-9656-11eb-832b-8499199c5342.png)

Witness report showing valid/invalid and based on distance:
![](https://user-images.githubusercontent.com/1075352/113659402-cb792480-9656-11eb-8f6c-3508c0a72275.png)

Drill down graph for two hotspots talking to each other:
![](https://user-images.githubusercontent.com/1075352/113659861-d1233a00-9657-11eb-9409-70af2c7d9d7b.png)

[Detailed graph information](GRAPHS.md).

## Installation

You can [grab a precompiled binary](
https://github.com/synfinatic/helium-analysis/releases) or build it yourself.

To build the binary:

 1. Make sure you have [Golang](https://www.golang.org) installed.
 1. Clone the repo
 1. Run: `make` using GNU Make (not BSD make).  Your binary will be placed in
    the `dist` directory.

## Running

helium-analysis has built in help for commands via the `-h` flag.  For the
most up-to-date information on how to use helium-analysis run 
`./helium-analysis -h`.

#### Commands

 * `graph` - Generate graphs for a hotspot
 * `hotspots` - Manage the hotspot cache
 * `challenges` - Manage the challenge data for hotspots
 * `names` - Show hotspot name to address mappings
 * `version` - Display version information 

#### Overview

helium-analysis uses a database (default file is `helium.db`) to store all of
hotspot and challenge data necessary to generate the graphs.  To avoid over-loading
the Helium API servers, graphs are generated only using the data stored in the
database.  In order to populate the database you should run:

 1. `helium-analysis hotspots refresh`  -- Load the metadata for all of the Helium 
        hotspots.
 1. `helium-analysis challenges refresh <address>` -- Load the challenges
        for a given hotspot. Warning: this can take 10 or more minutes!  Do not 
        interrupt the process or all downloaded data will be lost.
 1. `helium-analysis graph <address>` -- Generate graphs for the specified hotspot.

Note that you can specify the hotspot name OR address for the challenges and graph 
commands, but the address is recommended to avoid issues with name collisions.
## Donate

If you find this useful, feel free to throw a few HNT my way: `144xaKFbp4arCNWztcDbB8DgWJFCZxc8AtAKuZHZ6Ejew44wL8z`

## License

Helium Analysis is Licensed under the [GPLv3](https://www.gnu.org/licenses/gpl-3.0.en.html).
