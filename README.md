# Helium Analysis

Some simple tools for optmizing your [Helium](https://www.helium.com) hotspot
antenna and location.

## What?

Creates pretty graphs of PoC hotspot activity.  All the examples below
use the default 500 challenges worth of analysis.

Good example of two nodes talking to each other very consistently:
![](https://user-images.githubusercontent.com/1075352/112706374-f72f2a00-8e60-11eb-902e-cda4a2f7a4c5.png)

Notice the empty space on the right hand side indicating they haven't witnessed
each other for a few days.  Dots show actual PoC messages (both RX and TX
where appropriate) while the lines show the trailing average.
![](https://user-images.githubusercontent.com/1075352/112706137-7a4f8080-8e5f-11eb-9ef2-4dca63fccd6c.png)

Here is a graph showing more data points, including some invalid PoC witnesses.
![](https://user-images.githubusercontent.com/1075352/112706128-6ad03780-8e5f-11eb-943a-33b8ed942ecb.png)

Something clearly changed and better signal strength!
![](https://user-images.githubusercontent.com/1075352/112737511-4edc9c80-8f18-11eb-9327-96f420610b27.png)

Joined marker indicates when atomic-blood-woodpecker was added to the blockchain.
![](https://user-images.githubusercontent.com/1075352/112768602-9d01a680-8fd1-11eb-95f4-69e1481c4a70.png)

## Invalid?

Yes, if your signal strength is in the red, then it is considered invalid:
![](https://user-images.githubusercontent.com/1075352/112706552-2db97480-8e62-11eb-88d9-75b61af09279.png)

## Installation

You can [grab a precompiled binary](
https://github.com/synfinatic/helium-analysis/releases) or build it yourself.

To build the binary:

 1. Make sure you have [Golang](https://www.golang.org) installed.
 1. Clone the repo
 1. Run: `make` using GNU Make (not BSD make).  Your binary will be placed in
    the `dist` directory.

## Running

From a terminal, execute: `./helium-analysis --address XXXXXX` where XXXXXX
is the hotspot address you wish to analyze.  Optionally, you can use
`--name` to specify the hotspot name to analyze.  This will generate a PNG file
for each hotspot you witness and/or witnessed you.

Note that there is no GUI/WebUI at this time.

## Flags

 * `--address X` - Specify the hotspot address to analyze
 * `--name X` - Specify the hotspot name to analyze
 * `--zoom` - Unfix the X & Y axis and zoom in on each individual graph
 * `--min X` - Set the minimum of data points required to generate a graph  (deafult 5)
 * `--days X` - Set the number of days to process (default 30)
 * `--hotspots` - Refresh hotspots cache
 * `--no-cache` - Disable caching of challenges
 * `--expires X` - Refresh challenges if more than X hours old
 * `--version` - Print out the version information

## Donate

If you find this useful, feel free to throw a few HNT my way: `144xaKFbp4arCNWztcDbB8DgWJFCZxc8AtAKuZHZ6Ejew44wL8z`

## License

Helium Analysis is Licensed under the [GPLv3](https://www.gnu.org/licenses/gpl-3.0.en.html).
