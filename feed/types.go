package feed

import "time"

type Update struct {
	Price string
	Size  string
}

type orderbookSortedKey struct {
	Value float64
	Key   string
}

type sortByOrderbookPrice []*orderbookSortedKey

type LevelTwoOrderbook struct {
	Bids [][]interface{} `json:"bids"`
	Asks [][]interface{} `json:"asks"`
}

type TickerChannel struct {
	Name       string   `json:"name"`
	ProductIds []string `json:"product_ids"`
}

type WebsocketType struct {
	Type string `json:"type"`
}
type MessageSubscription struct {
	WebsocketType
	ProductIds []string      `json:"product_ids"`
	Channels   []interface{} `json:"channels"`
}
type L2UpdateMessage struct {
	WebsocketType
	ProductID string     `json:"product_id"`
	Changes   [][]string `json:"changes"`
	Time      time.Time  `json:"time"`
}

type L2SnapshotMessage struct {
	WebsocketType
	ProductID string     `json:"product_id"`
	Bids      [][]string `json:"bids"`
	Asks      [][]string `json:"asks"`
}

type OrderbookModel interface {
	SetSnapshot(epoch int64, bids []*Update, asks []*Update) bool
	WriteUpdate(epoch int64, bids []*Update, asks []*Update) bool
}
