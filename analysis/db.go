package analysis

/*
 * Helium Analysis
 * Copyright (c) 2021-2022 Aaron Turner  <aturner at synfin dot net>
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
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

// "constants" in go are awesome
var HOTSPOTS_BUCKET []byte = []byte("hotspots")
var HOTSPOT_NAMES_BUCKET []byte = []byte("hotspot_names")
var CHALLENGES_BUCKET []byte = []byte("challenges")
var HOTSPOTS_CACHE_KEY []byte = []byte("hotspotcache")
var META_BUCKET []byte = []byte("metadata")
var VERSION_KEY []byte = []byte("version")
var DB_VERSION []byte = []byte("v1")

const TIME_FORMAT = "2006-01-02 15:04:05 MST"

type BoltDB struct {
	db             *bolt.DB
	hotspotCache   map[string]Hotspot
	hotspotAddress map[string]string
}

func OpenDB(filename string, init bool) (*BoltDB, error) {
	fileInfo, err := os.Stat(filename)
	if os.IsNotExist(err) && !init {
		return nil, fmt.Errorf("Database '%s' does not exist.  Create new DB via --init-db", filename)
	} else if !init && fileInfo.IsDir() {
		return nil, fmt.Errorf("Can not use directory '%s' as database file.", filename)
	}

	x, err := bolt.Open(filename, 0666, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	b := BoltDB{
		db:             x,
		hotspotCache:   map[string]Hotspot{},
		hotspotAddress: map[string]string{},
	}

	// initialize
	err = b.db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists(HOTSPOTS_BUCKET)
		if err != nil {
			return fmt.Errorf("Uanble to create bucket: %s", string(HOTSPOTS_BUCKET))
		}

		_, err = tx.CreateBucketIfNotExists(HOTSPOT_NAMES_BUCKET)
		if err != nil {
			return fmt.Errorf("Uanble to create bucket: %s", string(HOTSPOT_NAMES_BUCKET))
		}

		_, err = tx.CreateBucketIfNotExists(CHALLENGES_BUCKET)
		if err != nil {
			return fmt.Errorf("Uanble to create bucket: %s", string(CHALLENGES_BUCKET))
		}

		meta, err := tx.CreateBucketIfNotExists(META_BUCKET)
		if err != nil {
			return fmt.Errorf("Uanble to create bucket: %s", string(META_BUCKET))
		}

		version := meta.Get(VERSION_KEY)
		if version == nil {
			// set version 1
			meta.Put(VERSION_KEY, DB_VERSION)
		} else if bytes.Compare(DB_VERSION, version) != 0 {
			log.Panicf("Database version miss-match. Expected %s, but is %s",
				string(DB_VERSION), string(version))
		}

		return nil
	})
	return &b, err
}

func (b *BoltDB) Close() {
	b.db.Close()
}

func (b *BoltDB) GetDb() *bolt.DB {
	return b.db
}

// Get the Hotspot metadata for a given address
func (b *BoltDB) GetHotspot(address string) (Hotspot, error) {
	h := Hotspot{}
	if h, ok := b.hotspotCache[address]; ok {
		return h, nil
	}

	err := b.db.View(func(tx *bolt.Tx) error {
		buck := tx.Bucket(HOTSPOTS_BUCKET)
		v := buck.Get([]byte(address))
		if v != nil {
			json.Unmarshal(v, &h)
		}
		return nil
	})
	b.hotspotCache[address] = h
	return h, err
}

// returns the height of the block chain when we cached the hotspots
func (b *BoltDB) GetHotspotsCacheHeight() (int64, error) {
	var val int64
	err := b.db.View(func(tx *bolt.Tx) error {
		buck := tx.Bucket(HOTSPOTS_BUCKET)
		cursor := buck.Cursor()
		_, v := cursor.First()
		hotspot := Hotspot{}
		json.Unmarshal(v, &hotspot)
		val = hotspot.Block
		return nil
	})
	return val, err
}

func (b *BoltDB) AutoRefreshHotspots(limit int64) error {
	height, err := GetCurrentHeight()
	if err != nil {
		return err
	}
	ourHeight, err := b.GetHotspotsCacheHeight()
	if err != nil {
		return err
	}
	if (height - ourHeight) > limit {
		log.Infof("Hotspot data is %d blocks old.  Refreshing...", (height - ourHeight))
		hotspots, err := FetchHotspots()
		if err != nil {
			return err
		}
		err = b.SetHotspots(hotspots)
	}
	return err
}

// Get all the hotspots in the DB
func (b *BoltDB) GetHotspots() ([]Hotspot, error) {
	hotspots := []Hotspot{}
	err := b.db.View(func(tx *bolt.Tx) error {
		buck := tx.Bucket(HOTSPOTS_BUCKET)
		cursor := buck.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			h := Hotspot{}
			if v != nil {
				json.Unmarshal(v, &h)
			}
			hotspots = append(hotspots, h)
		}
		return nil
	})
	return hotspots, err
}

// Write a list of hotspots to the database under the address and name
func (b *BoltDB) SetHotspots(hotspots []Hotspot) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		for _, hotspot := range hotspots {
			err := b.setHotspot(tx, hotspot)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

// Write a list of hotspots to the database under the address
func (b *BoltDB) SetAllHotspots(hotspots []Hotspot) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		for _, hotspot := range hotspots {
			err := b.setHotspot(tx, hotspot)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (b *BoltDB) setHotspot(tx *bolt.Tx, hotspot Hotspot) error {
	jdata, err := json.Marshal(hotspot)
	if err != nil {
		return err // rollback
	}

	bucket := tx.Bucket(HOTSPOTS_BUCKET)

	// store canonical value based on address
	err = bucket.Put([]byte(hotspot.Address), jdata)
	if err != nil {
		return err // rollback
	}

	namesBucket := tx.Bucket(HOTSPOT_NAMES_BUCKET)

	// store name => address mapping
	check := namesBucket.Get([]byte(hotspot.Name))
	if check == nil {
		err = namesBucket.Put([]byte(hotspot.Name), []byte(hotspot.Address))
		if err != nil {
			return err // rollback
		}
	}

	/*
		FIXME: not really sure what info I want to keep?  The whole record is quite large
		and only a few variables change very often.  Can we just store a diff?

		// store timeseries, using the block as the key
		tsbuck, err := tx.CreateBucketIfNotExists([]byte(hotspot.Address))
		if err != nil {
			return err // rollback
		}
		key := make([]byte, 8)
		binary.PutVarint(key, hotspot.Block)
		err = tsbuck.Put(key, jdata)
	*/

	b.hotspotCache[hotspot.Address] = hotspot
	return nil
}

// Lookup the hotspot name by address
func (b *BoltDB) GetHotspotName(address string) (string, error) {
	hotspot := Hotspot{}

	// check the cache
	if name, ok := b.hotspotAddress[address]; ok {
		return name, nil
	}
	err := b.db.View(func(tx *bolt.Tx) error {
		buck := tx.Bucket(HOTSPOTS_BUCKET)
		v := buck.Get([]byte(address))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &hotspot)
	})
	if err != nil {
		return "", err
	}
	// update cache if exists and return
	if hotspot.Name != "" {
		b.hotspotAddress[hotspot.Name] = address
	}
	return hotspot.Name, nil
}

// Lookup the hotspot address by name
func (b *BoltDB) GetHotspotAddress(name string) (string, error) {
	address := ""

	// check cache
	if h, ok := b.hotspotCache[address]; ok {
		return h.Name, nil
	}
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(HOTSPOT_NAMES_BUCKET)
		v := bucket.Get([]byte(name))
		if v == nil {
			return fmt.Errorf("%s is not in database", name)
		}
		if len(v) > 0 {
			address = string(v)
		} else {
			return fmt.Errorf("%s is not in database: '%s'", name, string(v))
		}
		return nil
	})
	return address, err
}

// Loads all the current hotspots into the database, if we are too old
func (b *BoltDB) LoadHotspots(last time.Time) error {

	now := time.Now().UTC()
	if last.Before(now) {
		return nil
	}

	hotspots, err := FetchHotspots()
	if err != nil {
		return err
	}

	return b.SetAllHotspots(hotspots)
}

// returns the challenge bucket for a hotspot address
func ChallengeBucket(tx *bolt.Tx, address string) (*bolt.Bucket, error) {
	x := strings.Split(address, "-")
	if len(x) == 3 {
		return nil, fmt.Errorf("Invalid address: %s", address)
	}

	bucket := tx.Bucket(CHALLENGES_BUCKET)
	challengeBucket, err := bucket.CreateBucketIfNotExists([]byte(address))
	if err != nil {
		return nil, err
	}

	return challengeBucket, nil
}

// Load all the challenges as old as first if last <= time.UTC()
func (b *BoltDB) LoadChallenges(address string, first, last time.Time, holddown time.Duration) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		// create a bucket for the challenges of our hotspot
		bucket, err := ChallengeBucket(tx, address)
		if err != nil {
			return err
		}

		cursor := bucket.Cursor()

		var kFirstTime, kLastTime time.Time
		// lookup first and last time in DB
		kLast, _ := cursor.Last()

		loadUntil := time.Unix(0, 0)

		// we have some data in the bucket, so see if it out of date enough
		if kLast != nil {
			bLast := binary.BigEndian.Uint64(kLast)
			kLastTime = time.Unix(int64(bLast), 0).UTC()

			kFirst, _ := cursor.First()
			bFirst := binary.BigEndian.Uint64(kFirst)
			kFirstTime = time.Unix(int64(bFirst), 0).UTC()
			log.Debugf("Database challenges: %s => %s",
				kFirstTime.Format(TIME_FORMAT), kLastTime.Format(TIME_FORMAT))

			// If the DB has challenges is +/- the holddown, then we're "good enough"
			// Note that this is the _challenge time_ not the last time we ran `refresh`
			lastSearch := last.Add(-holddown)
			firstSearch := first.Add(holddown)

			if kLastTime.Equal(time.Unix(0, 0)) {
				loadUntil = first.Add(holddown)
				log.Infof("No entries in database....")
			} else if kFirstTime.After(firstSearch) {
				loadUntil = first.Add(-holddown)
				t := firstSearch.Format(TIME_FORMAT)
				log.Infof("First database record is after %s", t)
			} else if kLastTime.Before(lastSearch) {
				loadUntil = kLastTime.Add(-holddown)
				t := lastSearch.Format(TIME_FORMAT)
				log.Infof("Last database record is before %s", t)
			}

			if loadUntil.Equal(time.Unix(0, 0)) {
				log.Infof("Cache is up to date for %s", address)
				return nil
			}
		} else {
			// load all the data for the graph
			loadUntil = first.Add(-holddown)
		}
		log.Debugf("Loading challenges until: %s", loadUntil.Format(TIME_FORMAT))

		// load everything we need from the API
		challenges, err := FetchChallenges(address, loadUntil)
		if err != nil {
			return err // rollback
		}

		// Add any new records
		cnt := 0
		for _, challenge := range challenges {
			t := time.Unix(challenge.Time, 0)

			if t.After(kLastTime) || t.Before(kFirstTime) {
				jdata, err := json.Marshal(challenge)
				if err != nil {
					log.WithError(err).Errorf("unable to marshall json")
					return err // rollback
				}
				key := make([]byte, 8)
				binary.BigEndian.PutUint64(key, uint64(challenge.Time))
				err = bucket.Put(key, jdata)
				cnt += 1
				if err != nil {
					log.WithError(err).Errorf("Unable to put %v => %s", key, string(jdata))
					tx.Rollback()
					return err
				}
			}
		}
		log.Infof("Loaded %d new challenges into database", cnt)
		return nil
	})
	return err
}

// returns a list of challenges for the given hotspot
func (b *BoltDB) GetChallenges(address string, first time.Time, last time.Time) ([]Challenges, error) {
	challenges := []Challenges{}

	err := b.db.Update(func(tx *bolt.Tx) error {
		buck, err := ChallengeBucket(tx, address)
		if err != nil {
			return err
		}
		cursor := buck.Cursor()
		minKey := make([]byte, 8)
		maxKey := make([]byte, 8)
		binary.BigEndian.PutUint64(minKey, uint64(first.Unix()))
		binary.BigEndian.PutUint64(maxKey, uint64(last.Unix()))

		for k, v := cursor.Seek(minKey); k != nil && bytes.Compare(k, maxKey) <= 0; k, v = cursor.Next() {
			challenge := Challenges{}
			err := json.Unmarshal(v, &challenge)
			if err != nil {
				return err
			}
			challenges = append(challenges, challenge)
		}
		return nil
	})
	return challenges, err
}

// Returns the address of the hotspot given an address or name
func (b *BoltDB) GetHotspotByUnknown(addressOrName string) (string, error) {
	var hotspotAddress string
	var err error

	x := strings.Split(addressOrName, "-")

	if len(x) == 3 {
		// user provided hotspot name
		hotspotAddress, err = b.GetHotspotAddress(addressOrName)
		if err != nil {
			log.WithError(err).Errorf("Unable to lookup: %s", addressOrName)
		}
		if err != nil || len(hotspotAddress) == 0 {
			hotspotAddress, err = b.GetHotspotAddress(addressOrName)
			if err != nil {
				return "", fmt.Errorf("Invalid hotspot name '%s'.  Refresh hotspot cache?", addressOrName)
			}
			log.Debugf("Found: %s", hotspotAddress)
		}
	} else if len(x) == 1 {
		_, err := b.GetHotspot(addressOrName)
		if err != nil {
			return "", fmt.Errorf("Invalid hotspot address '%s'.  Refresh hotspot cache?", addressOrName)
		}
		hotspotAddress = addressOrName
	} else {
		return "", fmt.Errorf("Invalid hotspot address: %s", addressOrName)
	}
	return hotspotAddress, nil
}
