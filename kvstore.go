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
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	"go.etcd.io/raft/v3/raftpb"
)

// a key-value store backed by raft
type rvstore struct {
	node        int
	proposeC    chan<- string // channel for proposing updates
	mu          sync.RWMutex
	currentRv   *rv
	rvs         chan *rv
	snapshotter *snap.Snapshotter
	walMu       sync.RWMutex
	wal         map[string]json.RawMessage
}

type resource struct {
	Key       string `json:"key"`
	Timestamp int64  `json:"timestamp"`
}

type rv struct {
	resource
	ResourceVersion uint64
}

type rawData struct {
	RV       uint64 `json:"rv"`
	resource `json:",inline"`
	Obj      json.RawMessage `json:"obj"`
}

func newRVStore(node int, snapshotter *snap.Snapshotter, proposeC chan<- string, commitC <-chan *commit, errorC <-chan error) *rvstore {
	f, err := os.OpenFile(fmt.Sprintf("raftexample-%d.jsonl", node), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Panic(err)
	}
	s := &rvstore{proposeC: proposeC, snapshotter: snapshotter, rvs: make(chan *rv, 1000), node: node, wal: make(map[string]json.RawMessage)}
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

	go s.watchRVs(json.NewEncoder(f))

	// read commits from raft into kvStore map until error
	go s.readCommits(commitC, errorC)
	return s
}

func (s *rvstore) watchRVs(encoder *json.Encoder) {
	for {
		rv := <-s.rvs
		s.walMu.RLock()
		obj, ok := s.wal[rv.Key]
		if !ok {
			// obj doesn't belong to node or has been committed
			s.walMu.RUnlock()
			continue
		}
		s.walMu.RUnlock()
		err := encoder.Encode(&rawData{RV: rv.ResourceVersion, resource: rv.resource, Obj: obj})
		if err != nil {
			log.Printf("Failed to write to file (%v)\n", err)
		}

		s.walMu.Lock()
		delete(s.wal, rv.Key)
		s.walMu.Unlock()
	}
}

func (s *rvstore) Current() (*rv, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.currentRv == nil {
		return nil, false
	}
	return s.currentRv, true
}

func (s *rvstore) Write(obj json.RawMessage) error {
	key := fmt.Sprintf("%d-%s", s.node, uuid.New().String())
	next := resource{Key: key, Timestamp: time.Now().UTC().UnixNano()}
	s.walMu.Lock()
	s.wal[key] = obj
	s.walMu.Unlock()
	buf := strings.Builder{}
	if err := gob.NewEncoder(&buf).Encode(next); err != nil {
		return err
	}
	s.proposeC <- buf.String()
	return nil
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
