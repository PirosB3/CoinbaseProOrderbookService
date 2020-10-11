package controller

import (
	"context"
	"errors"
	"pirosb3/real_feed/datasource"
	"pirosb3/real_feed/feed"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
)

const CHANNEL_BUFFER_SIZE = 20
const TS_LAYOUT = "2006-01-02T15:04:05.000000Z"

func DateStringToUnixEpoch(timestamp string) (int64, error) {
	t, err := time.Parse(TS_LAYOUT, timestamp)
	if err != nil {
		return -1, err
	}
	return int64(t.Unix()), nil
}

var (
	heartbeatTicker = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      "heartbeat",
		Help:      "Counts heartbeats from the websocket feed",
		Namespace: "feed",
	}, []string{"uuid", "market"})
)

type FeedController struct {
	orderbook *feed.OrderbookFeed
	websocket *datasource.CoinbaseProWebsocket
	ctx       context.Context
	startLock sync.Mutex
	stopFn    context.CancelFunc
	started   bool
	outChan   chan (map[string]interface{})
	inChan    chan (interface{})
	product   string
	uuid      string
}

func NewFeedController(
	ctx context.Context,
	product string,
) *FeedController {
	aUUID, _ := uuid.NewUUID()
	orderbook := feed.NewOrderbookFeed(product)
	newContext, stopFn := context.WithCancel(ctx)
	return &FeedController{
		uuid:      aUUID.String(),
		orderbook: orderbook,
		stopFn:    stopFn,
		ctx:       newContext,
		started:   false,
		outChan:   make(chan (map[string]interface{}), CHANNEL_BUFFER_SIZE),
		inChan:    make(chan (interface{}), CHANNEL_BUFFER_SIZE),
		product:   product,
	}
}

func (fc *FeedController) Start() error {
	if fc.started {
		return errors.New("Feed Controller is already started and cannot be restarted. Please create a new instance")
	}
	fc.startLock.Lock()
	defer fc.startLock.Unlock()

	fc.started = true
	fc.websocket = datasource.NewCoinbaseProWebsocket(fc.product, fc.outChan, fc.inChan, fc.ctx)
	fc.websocket.Start()

	go fc.runLoop()
	return nil
}

func (fc *FeedController) runLoop() {
	for {
		select {
		case <-fc.ctx.Done():
			log.Warning("Feed controller event loop shut down")
			return
		case wsType := <-fc.outChan:
			switch wsType["type"].(string) {
			case "snapshot":
				bidsInterface := wsType["bids"].([]interface{})
				bids := make([]*feed.Update, len(bidsInterface))
				for idx, bidsEl := range bidsInterface {
					bids[idx] = &feed.Update{
						Price: bidsEl.([]interface{})[0].(string),
						Size:  bidsEl.([]interface{})[1].(string),
					}
				}

				asksInterface := wsType["asks"].([]interface{})
				asks := make([]*feed.Update, len(asksInterface))
				for idx, asksEl := range asksInterface {
					asks[idx] = &feed.Update{
						Price: asksEl.([]interface{})[0].(string),
						Size:  asksEl.([]interface{})[1].(string),
					}
				}
				fc.orderbook.SetSnapshot(time.Now().Unix(), bids, asks)
				log.WithField("numBids", len(bids)).WithField("numAsks", len(asks)).Infoln("Set new snapshot")
			case "l2update":
				timestamp, err := DateStringToUnixEpoch(wsType["time"].(string))
				if err != nil {
					log.WithField("timestamp", wsType["time"].(string)).Errorln("Incorrect date format found.")
					continue
				}

				var bids []*feed.Update
				var asks []*feed.Update
				changes := wsType["changes"].([]interface{})
				for _, change := range changes {
					changeEl := change.([]interface{})
					update := &feed.Update{
						Price: changeEl[1].(string),
						Size:  changeEl[2].(string),
					}
					switch changeEl[0] {
					case "buy":
						bids = append(bids, update)
					case "sell":
						asks = append(asks, update)
					}
				}
				fc.orderbook.WriteUpdate(timestamp, bids, asks)
			case "heartbeat":
				heartbeatTicker.WithLabelValues(fc.uuid, fc.product).Inc()
			case "subscriptions":
			default:
				log.WithField("messageType", wsType["type"].(string)).Warningln("Received an unexpected message")
			}
		}
	}
}

func (fc *FeedController) Stop() {
	if fc.started {
		fc.stopFn()
	}
}

func (fc *FeedController) BuyQuote(amount float64) (float64, error) {
	return fc.orderbook.BuyQuote(amount)
}
func (fc *FeedController) SellQuote(amount float64) (float64, error) {
	return fc.orderbook.SellQuote(amount)
}
func (fc *FeedController) BuyBase(amount float64) (float64, error) {
	return fc.orderbook.BuyBase(amount)
}
func (fc *FeedController) SellBase(amount float64) (float64, error) {
	return fc.orderbook.SellBase(amount)
}
