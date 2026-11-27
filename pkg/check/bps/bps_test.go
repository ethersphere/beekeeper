package bps_test

import (
	"testing"

	"github.com/ethersphere/beekeeper/pkg/check/bps"
)

const (
	// a caller waiting in ReadMsgWithContext, and the reader it started
	waiter = `goroutine 10 [select]:
github.com/ethersphere/bee/v2/pkg/p2p/protobuf.Reader.ReadMsgWithContext({{0x1, 0x2}}, {0x3, 0x4}, {0x5, 0x6})
	github.com/ethersphere/bee/v2/pkg/p2p/protobuf/protobuf.go:72 +0x124
github.com/ethersphere/bee/v2/pkg/pullsync.(*Syncer).Sync(0x1)
	github.com/ethersphere/bee/v2/pkg/pullsync/pullsync.go:250 +0x3d0`
	liveReader = `goroutine 11 [select]:
github.com/libp2p/go-yamux/v5.(*Stream).Read(0x1, {0x2, 0x1000, 0x1000})
	github.com/libp2p/go-yamux/v5@v5.1.0/stream.go:113 +0x1b0
github.com/ethersphere/bee/v2/pkg/p2p/protobuf.Reader.ReadMsgWithContext.func1()
	github.com/ethersphere/bee/v2/pkg/p2p/protobuf/protobuf.go:65 +0x30
created by github.com/ethersphere/bee/v2/pkg/p2p/protobuf.Reader.ReadMsgWithContext in goroutine 10
	github.com/ethersphere/bee/v2/pkg/p2p/protobuf/protobuf.go:64 +0xcc`
	// a reader whose caller, goroutine 20, has returned
	orphanReader = `goroutine 21 [select, 3 minutes]:
github.com/libp2p/go-yamux/v5.(*Stream).Read(0x1, {0x2, 0x1000, 0x1000})
	github.com/libp2p/go-yamux/v5@v5.1.0/stream.go:113 +0x1b0
github.com/ethersphere/bee/v2/pkg/p2p/protobuf.Reader.ReadMsgWithContext.func1()
	github.com/ethersphere/bee/v2/pkg/p2p/protobuf/protobuf.go:65 +0x30
created by github.com/ethersphere/bee/v2/pkg/p2p/protobuf.Reader.ReadMsgWithContext in goroutine 20
	github.com/ethersphere/bee/v2/pkg/p2p/protobuf/protobuf.go:64 +0xcc`
	session = `goroutine 30 [select]:
github.com/libp2p/go-yamux/v5.(*Stream).Read(0x1, {0x2, 0x1, 0x1})
	github.com/libp2p/go-yamux/v5@v5.1.0/stream.go:113 +0x1b0
github.com/ethersphere/bee/v2/pkg/bps.(*session).readLoop.func1()
	github.com/ethersphere/bee/v2/pkg/bps/bps.go:357 +0x84`
	// the bridge waiting on its session, and its outbox reading the socket
	bridge = `goroutine 50 [chan receive]:
github.com/ethersphere/bee/v2/pkg/api.(*Service).bpsRun(0x1, 0x2, {{0x3, 0x4}})
	github.com/ethersphere/bee/v2/pkg/api/bps.go:368 +0x5a4
created by github.com/ethersphere/bee/v2/pkg/api.(*Service).bpsWsHandler in goroutine 49
	github.com/ethersphere/bee/v2/pkg/api/bps.go:130 +0x1c8`
	outbox = `goroutine 51 [IO wait]:
github.com/gorilla/websocket.(*Conn).ReadMessage(0x1)
	github.com/gorilla/websocket@v1.5.3/conn.go:1093 +0x24
github.com/ethersphere/bee/v2/pkg/api.bpsHandleOutbox({0x1, 0x2}, 0x3, {0x4, 0x5}, 0x6)
	github.com/ethersphere/bee/v2/pkg/api/bps.go:377 +0x90`
	// another handler of pkg/api, in a file whose name starts the same way
	apiOther = `goroutine 60 [select]:
github.com/ethersphere/bee/v2/pkg/api.(*Service).pssWsHandler(0x1, {0x2, 0x3}, 0x4)
	github.com/ethersphere/bee/v2/pkg/api/pss.go:60 +0x1c8`
	other = `goroutine 40 [chan receive]:
github.com/ethersphere/bee/v2/pkg/pusher.(*Service).chunksWorker(0x1)
	github.com/ethersphere/bee/v2/pkg/pusher/pusher.go:120 +0x1c`
)

func TestBpsGoroutines(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		stacks []string
		want   int
	}{
		{name: "empty dump"},
		{name: "live read of another protocol", stacks: []string{other, waiter, liveReader}},
		{name: "reader listed before its caller", stacks: []string{liveReader, waiter}},
		{name: "reader nobody waits for", stacks: []string{waiter, liveReader, orphanReader}, want: 1},
		{name: "bps code", stacks: []string{session, other}, want: 1},
		{name: "both", stacks: []string{session, orphanReader, waiter, liveReader}, want: 2},
		{name: "bridge", stacks: []string{bridge, outbox, other}, want: 2},
		{name: "another api handler", stacks: []string{apiOther, other}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var dump string
			for _, s := range tc.stacks {
				dump += s + "\n\n"
			}
			if got := bps.BpsGoroutines(dump); got != tc.want {
				t.Fatalf("got %d goroutines, want %d", got, tc.want)
			}
		})
	}
}
