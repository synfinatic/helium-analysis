# Helium Analysis Graphs

Helium analysis generates three types of graphs:

 1. Beacon report
 1. Witness report
 1. Peer hotspot report

## Beacon Report

Properly running hotspots should be challenged by other hotspots.  When
that happens, it sends a beacon out to be detected by other hotspots in 
the area.  This report shows how many witnesses each of your beacons 
had.  The dotted line shows a trailing average to help smooth the data
to be easier to understand.

![](https://user-images.githubusercontent.com/1075352/113659393-c6b47080-9656-11eb-832b-8499199c5342.png)


## Witness Report

This graph shows 30 days worth of witnesses for `fantastic-fiery-scallop`.
Witness are when other hotspots see your beacon.  Each green dot represents
a valid witness, while red invalid.  The X Axis represents time and the Y
Axis signal strength.

![](https://user-images.githubusercontent.com/1075352/113659402-cb792480-9656-11eb-8f6c-3508c0a72275.png)

## Peer Hotspot Report

For every nearby hotspot that sees your beacons or you witness, a graph will
be generated which looks something like this.  This graph has a little bit
of everything:

 1. A marker showing the time the other hotspot was added to the blockchain.
 1. The reward scale (0.50) of the remote hotspot as well as if it is online/offline.
 1. Distance in km and mi between the two hotspots.
 1. Red dashed line showing the max valid RSSI based on distance between the two
    hotspots.
 1. Orange dashed line showing the minimum valid RSSI based on the signal-to-noise
    ratio (SNR).
 1. Yellow dashed line showing the SNR for each beacon.
 1. Green dots and dashed lines showing the beacons signal strength sent by the 
    selected hotspot (early-mauve-cormorant) and recieved by the peer 
    (funny-seafoam-turtle) as well as the trailing average (dashed line). 
 1. Red dots show invalid beacons which have a signal outside of the limits.
 1. Blue dots and dashed lines showing the witness signal strength of beacons
    recieved by the selected hotspot from the peer.  
 1. Yellow dots show invalid witnesses which were outside of the limits.

![](https://user-images.githubusercontent.com/1075352/113659331-a684b180-9656-11eb-9321-e8e3c1a53563.png)

## Other Examples

wild-magenta-sardine has been offline for a few days now:

![](https://user-images.githubusercontent.com/1075352/113659882-dbddcf00-9657-11eb-8fe8-99599e66fed8.png)

Something seems to have changed which allows fantastic-fiery-scallop to have
started witnessing main-walnut-giraffe around 2021-03-20.

![](https://user-images.githubusercontent.com/1075352/113659861-d1233a00-9657-11eb-9409-70af2c7d9d7b.png)

An example of two hotspots which are not reliably witnessing each other because
the signal strength is invalid.

![](https://user-images.githubusercontent.com/1075352/113659746-902b2580-9657-11eb-9559-f95f86c57e52.png)
