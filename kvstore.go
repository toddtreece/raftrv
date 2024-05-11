// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	"go.etcd.io/raft/v3/raftpb"
)

// a key-value store backed by raft
type rvstore struct {
	proposeLock sync.Mutex
	node        int
	proposeC    chan<- string // channel for proposing updates
	mu          sync.RWMutex
	currentRv   *rv
	rvs         chan *rv
	snapshotter *snap.Snapshotter
}

type resource struct {
	Key       string
	Timestamp int64
}

type rv struct {
	resource
	ResourceVersion uint64
}

func newRVStore(node int, snapshotter *snap.Snapshotter, proposeC chan<- string, commitC <-chan *commit, errorC <-chan error) *rvstore {
	s := &rvstore{proposeC: proposeC, snapshotter: snapshotter, rvs: make(chan *rv, 1000), node: node}
	snapshot, err := s.loadSnapshot()
	if err != nil {
		log.Panic(err)
	}
	if snapshot != nil {
		log.Printf("loading snapshot at term %d and index %d", snapshot.Metadata.Term, snapshot.Metadata.Index)
		if err := s.recoverFromSnapshot(snapshot.Data); err != nil {
			log.Panic(err)
		}
	}
	// read commits from raft into kvStore map until error
	go s.readCommits(commitC, errorC)
	return s
}

func (s *rvstore) Current() (*rv, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.currentRv == nil {
		return nil, false
	}
	return s.currentRv, true
}

func (s *rvstore) propose(r *resource) error {
	var (
		buf strings.Builder
	)
	if err := gob.NewEncoder(&buf).Encode(r); err != nil {
		return err
	}
	s.proposeC <- buf.String()
	return nil
}

func (s *rvstore) Next(key string) (uint64, error) {
	s.proposeLock.Lock()
	defer s.proposeLock.Unlock()
	next := resource{Key: key, Timestamp: time.Now().UTC().UnixMilli()}
	resultCh := make(chan uint64, 1)
	go func(key string) {
		for rv := range s.rvs {
			if rv.Key == key {
				resultCh <- rv.ResourceVersion
				return
			}
		}
	}(key)
	s.propose(&next)

	return <-resultCh, nil
}

func (s *rvstore) readCommits(commitC <-chan *commit, errorC <-chan error) {
	for commit := range commitC {
		if commit == nil {
			// signaled to load snapshot
			snapshot, err := s.loadSnapshot()
			if err != nil {
				log.Panic(err)
			}
			if snapshot != nil {
				log.Printf("loading snapshot at term %d and index %d", snapshot.Metadata.Term, snapshot.Metadata.Index)
				if err := s.recoverFromSnapshot(snapshot.Data); err != nil {
					log.Panic(err)
				}
			}
			continue
		}

		for _, data := range commit.data {
			var dataResource resource
			dec := gob.NewDecoder(bytes.NewBufferString(data.data))
			if err := dec.Decode(&dataResource); err != nil {
				log.Fatalf("raftrv: could not decode message (%v)", err)
			}
			dataRv := &rv{resource: dataResource, ResourceVersion: data.index}
			s.mu.Lock()
			s.currentRv = dataRv
			s.rvs <- dataRv
			s.mu.Unlock()
		}
		close(commit.applyDoneC)
	}
	if err, ok := <-errorC; ok {
		log.Fatal(err)
	}
}

func (s *rvstore) getSnapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.Marshal(s.currentRv)
}

func (s *rvstore) loadSnapshot() (*raftpb.Snapshot, error) {
	snapshot, err := s.snapshotter.Load()
	if err == snap.ErrNoSnapshot {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (s *rvstore) recoverFromSnapshot(snapshot []byte) error {
	var store *rv
	if err := json.Unmarshal(snapshot, &store); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentRv = store
	return nil
}
