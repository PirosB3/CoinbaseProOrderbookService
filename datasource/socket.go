package datasource

import (
	"context"
	"errors"
	"net/http"
	"pirosb3/real_feed/feed"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
)

const heartbeatTTLSeconds = 4

var (
	pricingProm = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "pricing",
		Help:      "Orderbook visualization for 50 ETH",
		Namespace: "feed",
	}, []string{"uuid", "market"})

	updatesCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      "updates",
		Help:      "Shows the frequency of orderbook updates coming out of the websocket",
		Namespace: "feed",
	}, []string{"uuid", "market"})

	droppedPacketsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      "droppedPackets",
		Help:      "Shows the amount of dropped packets",
		Namespace: "feed",
	}, []string{"uuid", "market"})

	timeoutsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      "timeout",
		Help:      "Shows the frequency of timeouts",
		Namespace: "feed",
	}, []string{"uuid", "market"})
	wsLatency = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name:      "websocketUpdateFrequency",
		Help:      "Shows the frequency of websocket responses",
		Namespace: "feed",
	}, []string{"uuid", "market"})
)

type CoinbaseProWebsocket struct {
	uuid                string
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

// NewCoinbaseProWebsocket creates a new Coinbase Pro websocket feed. The feed will only start running once `.Start()` is called on the websocket.
// The `product` should be a Coinbase Pro ticket (example: "ETH-USD"), the `outChan` and `inChan` passed in allow the feed to send / receive messages
// and allow any external component to interact with the websocket service.
// This websocket is also fault-tolerant, if an update is not received within `heartbeatTTLSeconds` seconds, the websocket is automatically re-created.
// To shutdown the websocket, simply cancel the context passed in as first argument.
func NewCoinbaseProWebsocket(
	ctx context.Context,
	product string,
	outChan chan (map[string]interface{}),
	inChan chan (interface{}),
) *CoinbaseProWebsocket {
	aUUID, _ := uuid.NewUUID()
	return &CoinbaseProWebsocket{
		uuid:                aUUID.String(),
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
			// Some other process is trying to write a message to the websocket
			if ws.websocketConn == nil {
				log.Fatalln("Configured websocket does not exist, this should never happen. Message was skipped")
			}
			ws.websocketConn.WriteJSON(msgIn)
		case msgOut := <-ws.outInternalChan:
			// A message should be broadcasted to the outside. Writes the message to an outbound queue without blocking
			updatesCounter.WithLabelValues(ws.uuid, ws.product).Inc()
			select {
			case ws.outChan <- msgOut:
			default:
				log.Warningln("Websocket has no consumer for outgoing messages, dropping the message.")
				droppedPacketsCounter.WithLabelValues(ws.uuid, ws.product).Inc()
			}
		case <-time.After(time.Second * heartbeatTTLSeconds):
			// Something is wrong, websocket has not been responding for a fair amount of time. We should recreate the websocket
			timeoutsCounter.WithLabelValues(ws.uuid, ws.product).Inc()
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
		start := time.Now().Unix()
		var wsType map[string]interface{}
		err := connection.ReadJSON(&wsType)
		if err != nil {
			log.Errorln(err.Error())
			ws.websocketConn = nil
			return
		}
		ws.outInternalChan <- wsType
		end := time.Now().Unix()
		wsLatency.WithLabelValues(ws.uuid, ws.product).Observe(float64(end - start))
	}
}

// Start starts running the underlying websocket service. The function call does not block but
// it starts a series of underlying goroutines that are respoonsible for handling the websockets.
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
