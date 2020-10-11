package datasource

import (
	"context"
	"errors"
	"net/http"
	"pirosb3/real_feed/feed"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type CoinbaseProWebsocket struct {
	startLock           sync.Mutex
	websocketConn       *websocket.Conn
	product             string
	running             bool
	ctx                 context.Context
	outChan             chan (map[string]interface{})
	inChan              chan (interface{})
	outInternalChan     chan (map[string]interface{})
	timeoutInternalChan chan (bool)
}

func NewCoinbaseProWebsocket(product string, outChan chan (map[string]interface{}), inChan chan (interface{}), ctx context.Context) *CoinbaseProWebsocket {
	return &CoinbaseProWebsocket{
		product:             product,
		running:             false,
		ctx:                 ctx,
		inChan:              inChan,
		outChan:             outChan,
		outInternalChan:     make(chan (map[string]interface{})),
		timeoutInternalChan: make(chan bool),
	}
}

func (ws *CoinbaseProWebsocket) makeSubscriptionMessage() feed.MessageSubscription {
	subscription := feed.MessageSubscription{
		WebsocketType: feed.WebsocketType{
			Type: "subscribe",
		},
		ProductIds: []string{
			ws.product,
		},
		Channels: []interface{}{
			"level2",
			"heartbeat",
			feed.TickerChannel{
				Name: "ticker",
				ProductIds: []string{
					ws.product,
				},
			},
		},
	}
	return subscription
}

func (ws *CoinbaseProWebsocket) runLoop() {
	for {
		select {
		case <-ws.ctx.Done():
			// Parent context wants us to shut down. Simply stop websocket
			ws.timeoutInternalChan <- true
			return
		case msgIn := <-ws.inChan:
			if ws.websocketConn == nil {
				log.Fatalln("Configured websocket does not exist, this should never happen. Message was skipped")
			}
			ws.websocketConn.WriteJSON(msgIn)
		case msgOut := <-ws.outInternalChan:
			// A message should be broadcasted to the outside.
			select {
			case ws.outChan <- msgOut:
			default:
				log.Warningln("Websocket has no consumer for outgoing messages, dropping the message.")
			}
		case <-time.After(time.Second * 4):
			// Something is wrong, websocket has not been responding for a fair amount of time. We should recreate the websocket
			ws.timeoutInternalChan <- true
			go ws.setupWebsocket()
		}
	}
}

func (ws *CoinbaseProWebsocket) setupWebsocket() {
	var connection *websocket.Conn
	go func() {
		for {
			<-ws.timeoutInternalChan
			log.Warningln("Connection was intentionally closed due to a timeout or due to parent context closing")
			if connection != nil {
				connection.Close()
			}
			return
		}
	}()

	connection, _, err := websocket.DefaultDialer.Dial("wss://ws-feed.pro.coinbase.com", http.Header{})
	if err != nil {
		log.WithField("err", err.Error()).Errorln("error in dialling initial connection")
		return
	}
	ws.websocketConn = connection
	connection.WriteJSON(ws.makeSubscriptionMessage())
	for {
		var wsType map[string]interface{}
		err := connection.ReadJSON(&wsType)
		if err != nil {
			log.Errorln(err.Error())
			ws.websocketConn = nil
			return
		}
		ws.outInternalChan <- wsType
	}
}

func (ws *CoinbaseProWebsocket) Start() error {
	ws.startLock.Lock()
	defer ws.startLock.Unlock()

	if ws.running {
		return errors.New("Websocket was already running. Cancel the context for the websocket to close down")
	}
	ws.running = true

	// Start websocket internal component
	go ws.setupWebsocket()

	// Start context manager
	go ws.runLoop()

	return nil
}
