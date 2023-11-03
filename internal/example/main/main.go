// Copyright 2020 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/envoyproxy/go-control-plane/internal/example"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/envoyproxy/go-control-plane/pkg/test/v3"
)

var (
	l      example.Logger
	port   uint
	nodeID string
)

func init() {
	l = example.Logger{}

	flag.BoolVar(&l.Debug, "debug", false, "Enable xDS server debug logging")

	// The port that this xDS server listens on
	flag.UintVar(&port, "port", 1025, "xDS management server port")

	// Tell Envoy to use this Node ID
	flag.StringVar(&nodeID, "nodeID", "test-id", "Node ID")
}

func main() {
	flag.Parse()

	// Create a cache
	cache := cache.NewSnapshotCache(false, cache.IDHash{}, l)

	max_entries := 10000
	snap_id := 0

	// Create the snapshot that we'll serve to Envoy
	snapshot := example.GenerateSnapshot(snap_id, max_entries)
	if err := snapshot.Consistent(); err != nil {
		l.Errorf("snapshot inconsistency: %+v\n%+v", snapshot, err)
		os.Exit(1)
	}
	l.Debugf("will serve snapshot %+v", snapshot)

	// Add the snapshot to the cache
	if err := cache.SetSnapshot(context.Background(), nodeID, snapshot); err != nil {
		l.Errorf("snapshot error %q for %+v", err, snapshot)
		os.Exit(1)
	}

	go func() {
		time.Sleep(time.Second)
		stdin := bufio.NewReader(os.Stdin)
		for {
			fmt.Println("Type 'listener id <space> new route name' to trigger a change")
			var lid int
			var rname string
			n, err := fmt.Fscan(stdin, &lid, &rname)
			if err != nil || n != 2 || lid >= max_entries {
				l.Errorf("Fscan failed: n=%v err=%v lid=%v\n", n, err, lid)
				os.Exit(1)
			}

			fmt.Printf("Changing listener %v to use route name %v\n", lid, rname)
			example.ChangeRouteName(lid, rname)

			snap_id += 1
			snapshot := example.GenerateSnapshot(snap_id, max_entries)
			if err := snapshot.Consistent(); err != nil {
				l.Errorf("snapshot inconsistency: %+v\n%+v", snapshot, err)
				os.Exit(1)
			}

			if err := cache.SetSnapshot(context.Background(), nodeID, snapshot); err != nil {
				l.Errorf("snapshot error %q for %+v", err, snapshot)
				os.Exit(1)
			}
		}
	}()

	// Run the xDS server
	ctx := context.Background()
	cb := &test.Callbacks{Debug: l.Debug}
	srv := server.NewServer(ctx, cache, cb)
	example.RunServer(srv, port)
}
