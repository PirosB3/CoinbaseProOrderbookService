package feed

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	INSUFFICIENT_LIQUIDITY = "INSUFFICIENT_LIQUIDITY"
	BIDS                   = "BIDS"
	ASKS                   = "ASKS"
)

type OrderbookFeed struct {
	ProductId                string
	bids, asks               sortByOrderbookPrice
	bidsSizeMap, asksSizeMap map[string]float64
	lastEpochSeen            int64
	updateLock               *sync.Mutex
}

func (of *OrderbookFeed) GetProduct() (string, string) {
	items := strings.Split(of.ProductId, "-")
	if len(items) != 2 {
		panic("Expected 2 items")
	}
	return items[0], items[1]
}

func (of *OrderbookFeed) BuyQuote(amount float64) (float64, error) {
	remaining := amount
	baseAmountToPay := 0.0
	for _, orderSet := range of.bids {

		if remaining <= 0 {
			return baseAmountToPay, nil
		}

		size, ok := of.bidsSizeMap[orderSet.Key]
		if !ok {
			log.WithField("key", orderSet.Key).Errorln("Key cannot be found in lookup table.")
			continue
		}
		maxQuoteAmount := orderSet.Value * size
		amountToPurchase := maxQuoteAmount
		if amountToPurchase > remaining {
			amountToPurchase = remaining
		}

		// Perform the transaction
		remaining -= amountToPurchase
		baseAmountToPay += amountToPurchase / orderSet.Value
	}
	return -1, errors.New(INSUFFICIENT_LIQUIDITY)
}

func (of *OrderbookFeed) BuyBase(amount float64) (float64, error) {
	return of.performMarketOperation(amount, of.asks, of.asksSizeMap)
}

func (of *OrderbookFeed) SellBase(amount float64) (float64, error) {
	return of.performMarketOperation(amount, of.bids, of.bidsSizeMap)
}

func (of *OrderbookFeed) performMarketOperation(amount float64, book sortByOrderbookPrice, sizeMap map[string]float64) (float64, error) {
	if amount <= 0 {
		return -1, errors.New("Amount invalid")
	}
	remainingAmt := amount
	profitMade := 0.0
	for _, orderSet := range book {

		orderSize := sizeMap[orderSet.Key]
		amountToConsume := orderSize
		if remainingAmt <= amountToConsume {
			amountToConsume = remainingAmt
		}
		remainingAmt -= amountToConsume
		profitMade += amountToConsume * orderSet.Value

		if remainingAmt < 0 {
			return -1, errors.New("Implementation error")
		}

		if remainingAmt == 0 {
			log.Infoln(profitMade)
			return profitMade, nil
		}
	}
	return -1, errors.New(INSUFFICIENT_LIQUIDITY)
}

func (of *OrderbookFeed) writeUpdate(updates []*Update, side string) bool {
	var selectedBookPtr *sortByOrderbookPrice
	var selectedMap map[string]float64
	if side == BIDS {
		selectedBookPtr = &of.bids
		selectedMap = of.bidsSizeMap
	} else if side == ASKS {
		selectedBookPtr = &of.asks
		selectedMap = of.asksSizeMap
	} else {
		panic("Unsupported side: " + side)
	}

	performedInsert := false
	for _, update := range updates {
		parsedSize, err := strconv.ParseFloat(update.Size, 64)
		if err != nil {
			log.WithField("msg", err.Error()).Errorln("Skipped update due to error")
			continue
		}
		_, ok := selectedMap[update.Price]
		selectedMap[update.Price] = parsedSize
		if !ok {
			parsedPrice, err := strconv.ParseFloat(update.Price, 64)
			if err != nil {
				log.WithField("msg", err.Error()).Errorln("Skipped update due to error")
				continue
			}
			*selectedBookPtr = append(*selectedBookPtr, &orderbookSortedKey{
				Key:   update.Price,
				Value: parsedPrice,
			})
			performedInsert = true
		}
	}
	return performedInsert
}

func (of *OrderbookFeed) GetBookCount() (int, int) {
	return len(of.bids), len(of.asks)
}

func (of *OrderbookFeed) setData(epoch int64, bids []*Update, asks []*Update, recreate bool) bool {
	// Initiate a lock
	of.updateLock.Lock()
	defer of.updateLock.Unlock()

	if epoch < of.lastEpochSeen {
		return false
	}

	if recreate {
		// Re-create all maps and structs
		of.bids = nil
		of.asks = nil
		of.asksSizeMap = make(map[string]float64)
		of.bidsSizeMap = make(map[string]float64)
	}

	// Write a fresh batch of updates
	containsNewInsertsBids := of.writeUpdate(bids, BIDS)
	containsNewInsertsAsks := of.writeUpdate(asks, ASKS)

	// Sort the results after the update was written
	if containsNewInsertsBids {
		sort.Slice(of.bids, func(i, j int) bool {
			return of.bids[i].Value > of.bids[j].Value
		})
	}
	if containsNewInsertsAsks {
		sort.Slice(of.asks, func(i, j int) bool {
			return of.bids[i].Value < of.bids[j].Value
		})
	}
	return true
}

func (of *OrderbookFeed) SetSnapshot(epoch int64, bids []*Update, asks []*Update) bool {
	return of.setData(epoch, bids, asks, true)
}

func (of *OrderbookFeed) WriteUpdate(epoch int64, bids []*Update, asks []*Update) bool {
	return of.setData(epoch, bids, asks, false)
}

func NewOrderbookFeed(productId string) *OrderbookFeed {
	return &OrderbookFeed{
		ProductId:     productId,
		lastEpochSeen: -1,
		updateLock:    &sync.Mutex{},
		asksSizeMap:   make(map[string]float64),
		bidsSizeMap:   make(map[string]float64),
	}
}
