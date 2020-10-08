package main

import (
	"encoding/json"
	"net/http"
	"pirosb3/real_feed/feed"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const URL = "https://api.pro.coinbase.com/products/ETH-DAI/book?level=2"

type LevelTwoOrderbook struct {
	Bids [][]interface{} `json:"bids"`
	Asks [][]interface{} `json:"asks"`
}

func TransformToUpdate(input [][]interface{}) []*feed.Update {
	newUpdates := make([]*feed.Update, len(input))
	for idx, val := range input {
		newUpdates[idx] = &feed.Update{
			Price: val[0].(string),
			Size:  val[1].(string),
		}
	}
	return newUpdates
}

func main() {
	response, err := http.Get(URL)
	defer response.Body.Close()
	if err != nil {
		panic(err)
	}

	var l2Data LevelTwoOrderbook
	decoder := json.NewDecoder(response.Body)
	decoder.Decode(&l2Data)

	ob := feed.NewOrderbookFeed("ETH-DAI")
	bids := TransformToUpdate(l2Data.Bids)
	asks := TransformToUpdate(l2Data.Asks)
	ob.SetSnapshot(time.Now().Unix(), bids, asks)

	done := make(chan bool)
	c, _, err := websocket.DefaultDialer.Dial("wss://ws-feed.pro.coinbase.com", http.Header{})
	if err != nil {
		log.Fatal("dial:", err)
	}
	go func() {
		for {
			var wsType map[string]interface{}
			err := c.ReadJSON(&wsType)
			if err != nil {
				log.Println("read:", err)
				return
			}
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
				ob.SetSnapshot(time.Now().Unix(), bids, asks)
				log.WithField("numBids", len(bids)).WithField("numAsks", len(asks)).Infoln("Set new snapshot")
			case "l2update":
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
				ob.WriteUpdate(time.Now().Unix(), bids, asks)
				// log.WithField("numBids", len(bids)).WithField("numAsks", len(asks)).Infoln("Performed L2 update")
			}
		}
	}()
	subscription := feed.MessageSubscription{
		WebsocketType: feed.WebsocketType{
			Type: "subscribe",
		},
		ProductIds: []string{
			"ETH-USD",
		},
		Channels: []interface{}{
			"level2",
			"heartbeat",
			feed.TickerChannel{
				Name: "ticker",
				ProductIds: []string{
					"ETH-USD",
				},
			},
		},
	}
	c.WriteJSON(&subscription)

	go func() {
		ticker := time.NewTicker(time.Second * 10)
		for {
			select {
			case <-ticker.C:
				quoteAmount, err := ob.BuyBase(50)
				if err != nil {
					log.WithField("err", err.Error()).Errorln("Error performing trade")
				}
				log.WithField("baseAmount", 50).WithField("quoteAmount", quoteAmount).Infoln("Buy base")
			}
		}
	}()
	<-done
}
