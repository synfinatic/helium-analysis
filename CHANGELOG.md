# Helium Analysis Changelog

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
