# Helium Analysis Changelog

## v0.9.0 - 2021-04-08

- Complete CLI revamp
- Switch to using an embedded database (single file) instead of JSON files
- Graphs are now placed in a directory named after the hotspot name
- Update README with new versions of the graphs
- Add GRAPHS.md
- Various tweaks to the graphs

## v0.8.0 - 2021-04-01

- Add online status & reward scale of remote node
- Connect avg lines when there is invalid PoC
- Code cleanup & refactor
- Use hotspot name as challenges cache file instead of just challenges.json
- Update Readme
- Add new beacons & witnesses graphs

## v0.7.1 - 2021-03-30

- Disable --zoom flag as it is broken
- Improve labels to include count of datapoints
- Switch to alternate colors instead of labels for invalid PoC
- Fix bug where only TX data was graphed
- Reduce size of data points from 5px to 3px

## v0.7.0 - 2021-03-30

- Remove --hotspots flag
- Add --force-cache flag to always use cache
- Add SNR data to the graph
- Add max valid witness RSSI strength to the graph
- Fix bug where other witness were incorrectly included in the graphs
- Add --json flag to dump per-hotspot challenge info
- Add hash to witness cache for debugging purposes

## v0.6.1 - 2021-03-29 

- Fix graphing of invalid PoC.  Now we use the `is_valid` flag provided
    by the witness data.
- TX/RX now uses consistent green & blue colors and dashed line for avg

## v0.6.0 - 2021-03-28

- Replace --challenges flag with --days
- Improve how the challenges.json cache is invalided for better performance

## v0.5.4 - 2021-03-28

- Fix challenges.json cache invalidation when switching hotspot to graph 
- Fix null pointer bug when getting challenge timestamp 
- Calculate timestamp more consistently

## v0.5.3 - 2021-03-28

- Add `Joined` marker for when a hotspot is added to the blockchain
- Add `--name` flag so you don't have to lookup the hotspot address

## v0.5.2 - 2021-03-27

- Auto-unlock Y axis if PoC are out of range
- Limit `--min` to be >= 2 to avoid deadlock
- Add missing copyright statement in code
- Improve readme

## v0.5.1 - 2021-03-27

- Auto-fetch hotspots so you don't have to use `--hotspots`

## v0.5.0 - 2021-03-27 [initial release]

- Cache challenges.json and add some basic invalidation
- Create a real readme

## <= v0.4.0

Unreleased
