package datasource

import (
	"context"
	"testing"
	"time"
)

func TestContextShutsDown(t *testing.T) {
	outChan := make(chan (map[string]interface{}))
	inChan := make(chan (interface{}))

	ctx, cancelFn := context.WithCancel(context.Background())
	ws := NewCoinbaseProWebsocket(
		ctx, "ETH-USD", outChan, inChan,
	)
	ws.Start()
	<-outChan
	if ws.websocketConn == nil {
		t.Error("Websocket was supposed to exist")
	}
	cancelFn()
	time.Sleep(time.Millisecond * 20)
	if ws.websocketConn != nil {
		t.Error("Cancel should have cleared up websocket context")
	}
}
