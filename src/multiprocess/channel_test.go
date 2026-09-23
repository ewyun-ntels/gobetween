package multiprocess

import (
	"net"
	"testing"

	"github.com/yyyar/gobetween/stats"
)

func TestChannelRoundTrip(t *testing.T) {
	left, right := net.Pipe()
	parent := newChannel(left)
	worker := newChannel(right)
	defer parent.Close()
	defer worker.Close()

	want := Message{
		Type:     messageStats,
		WorkerID: 2,
		PID:      1234,
		Stats: map[string]stats.Stats{
			"dns": {RxTotal: 10, TxTotal: 20},
		},
	}

	sendResult := make(chan error, 1)
	go func() { sendResult <- parent.Send(want) }()
	got, err := worker.Receive()
	if err != nil {
		t.Fatalf("Receive() error = %v", err)
	}
	if err := <-sendResult; err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if got.Type != want.Type || got.WorkerID != want.WorkerID || got.PID != want.PID {
		t.Fatalf("Receive() = %#v, want %#v", got, want)
	}
	if got.Stats["dns"].RxTotal != 10 || got.Stats["dns"].TxTotal != 20 {
		t.Fatalf("Receive() stats = %#v", got.Stats)
	}
}
