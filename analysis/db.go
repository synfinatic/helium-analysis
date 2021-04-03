package analysis

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
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

// "constants" in go are awesome
var HOTSPOTS_BUCKET []byte = []byte("hotspots")
var HOTSPOTS_CACHE_KEY []byte = []byte("hotspotcache")

type BoltDB struct {
	db             *bolt.DB
	hotspotCache   map[string]Hotspot
	hotspotAddress map[string]string
}

func OpenDB(filename string) (*BoltDB, error) {
	x, err := bolt.Open(filename, 0666, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	b := BoltDB{
		db: x,
	}

	// initialize
	err = b.db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists(HOTSPOTS_BUCKET)
		return err
	})
	return &b, nil
}

func (bolt *BoltDB) Close() {
	bolt.db.Close()
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

// Write a list of hotspots to the database under the address and name
func (b *BoltDB) SetHotspots(hotspots []Hotspot) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		buck := tx.Bucket(HOTSPOTS_BUCKET)
		for _, hotspot := range hotspots {
			b.setHotspot(buck, hotspot)
		}
		return tx.Commit()
	})
	return err
}

// Write a list of hotspots to the database under the address and name
// and update cache time
func (b *BoltDB) SetAllHotspots(hotspots []Hotspot) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		buck := tx.Bucket(HOTSPOTS_BUCKET)
		for _, hotspot := range hotspots {
			b.setHotspot(buck, hotspot)
		}
		buf := make([]byte, 8)
		binary.PutVarint(buf, time.Now().Unix())

		buck.Put(HOTSPOTS_CACHE_KEY, buf)
		return tx.Commit()
	})
	return err
}

func (b *BoltDB) setHotspot(buck *bolt.Bucket, hotspot Hotspot) error {
	jdata, err := json.Marshal(hotspot)
	if err != nil {
		return err // rollback
	}

	// store canonical value based on address
	err = buck.Put([]byte(hotspot.Address), jdata)
	if err != nil {
		return err // rollback
	}

	// store timeseries
	key := fmt.Sprintf("%s-%d", hotspot.Address, time.Now().Unix())
	err = buck.Put([]byte(key), jdata)
	if err != nil {
		return err // rollback
	}

	// store name => address mapping
	err = buck.Put([]byte(hotspot.Name), []byte(hotspot.Address))
	if err != nil {
		return err // rollback
	}
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
		buck := tx.Bucket(HOTSPOTS_BUCKET)
		v := buck.Get([]byte(name))
		if v == nil {
			return nil
		}
		if len(v) > 0 {
			address = string(v)
		} else {
			return fmt.Errorf("%s is not in database", name)
		}
		return nil
	})
	return address, err
}

// Loads all the current hotspots into the database, if we are too old
func (b *BoltDB) LoadHotspots(last time.Time) error {

	now := time.Now()
	if last.Before(now) {
		return nil
	}

	hotspots, err := FetchHotspots()
	if err != nil {
		return err
	}

	return b.SetAllHotspots(hotspots)
}

// Load all the challenges as old as first if last <= time.Now()
func (b *BoltDB) LoadChallenges(address string, first time.Time, last time.Time) error {
	bucketName := []byte(fmt.Sprintf("challenges:%s", address))
	now := time.Now()
	err := b.db.Update(func(tx *bolt.Tx) error {
		buck, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}

		cursor := buck.Cursor()

		// lookup first and last time in DB
		kLast, _ := cursor.Last()
		bLast, cnt := binary.Varint(kLast)
		if cnt < 0 {
			return fmt.Errorf("unable to decode kLast: %d", cnt)
		}
		kLastTime := time.Unix(bLast, 0)

		if kLastTime.Before(now) || kLastTime.Equal(now) {
			log.Infof("Cache is up to date")
			return nil
		} else {
			log.Infof("Updating challenges for %s", address)
		}

		kFirst, _ := cursor.First()
		bFirst, cnt := binary.Varint(kFirst)
		if cnt < 0 {
			return fmt.Errorf("unable to decode kFirst: %d", cnt)
		}

		kFirstTime := time.Unix(bFirst, 0)

		// see if we have everything already cached
		if (kFirstTime.Before(first) || kFirstTime.Equal(first)) &&
			(kLastTime.After(last) || kLastTime.Equal(last)) {
			return nil
		}

		// load everything we need from the API
		challenges, err := FetchChallenges(address, first)
		if err != nil {
			return err // rollback
		}

		// Add any new records
		for _, challenge := range challenges {
			t := time.Unix(challenge.Time, 0)

			if err != nil {
				return err // rollback
			}

			if t.After(kLastTime) || t.Before(kFirstTime) {
				jdata, err := json.Marshal(challenge)
				if err != nil {
					return err // rollback
				}
				buf := make([]byte, 8)
				binary.PutVarint(buf, challenge.Time)
				buck.Put(buf, jdata)
			}
		}

		return nil
	})
	return err
}

func (b *BoltDB) GetChallenges(address string, first time.Time, last time.Time) ([]Challenges, error) {
	bucketName := []byte(fmt.Sprintf("challenges:%s", address))
	challenges := []Challenges{}

	err := b.db.View(func(tx *bolt.Tx) error {
		buck, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}

		cursor := buck.Cursor()
		minKey := make([]byte, 8)
		maxKey := make([]byte, 8)
		binary.PutVarint(minKey, first.Unix())
		binary.PutVarint(maxKey, first.Unix())

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
