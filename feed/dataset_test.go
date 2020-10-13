package feed

import (
	"encoding/json"
	"math"
	"net/http"
	"testing"
	"time"
)

func transformToUpdate(input [][]interface{}) []*Update {
	newUpdates := make([]*Update, len(input))
	for idx, val := range input {
		newUpdates[idx] = &Update{
			Price: val[0].(string),
			Size:  val[1].(string),
		}
	}
	return newUpdates
}

func TestCanInitialize(t *testing.T) {
	ob := NewOrderbookFeed("ETH-DAI")
	base, quote := ob.GetProduct()
	if base != "ETH" || quote != "DAI" {
		t.Errorf("Expected ETH-DAI, instead got %s-%s", base, quote)
	}
}

func TestFailsForGetPrice(t *testing.T) {
	ob := NewOrderbookFeed("ETH-DAI")
	ob.SetSnapshot(time.Now().Unix(), []*Update{}, []*Update{})
	_, _, err := ob.SellBase(1.2)
	if err == nil {
		t.Error("Expected error to exist, but it was nil")
	}
	if err.Error() != INSUFFICIENT_LIQUIDITY {
		t.Errorf("Error message is incorrect, was %s", err.Error())
	}
}

func TestAddItem(t *testing.T) {
	ob := NewOrderbookFeed("ETH-DAI")
	bids := []*Update{
		&Update{Price: "333.2", Size: "0.5"},
		&Update{Price: "320", Size: "0.5"},
		&Update{Price: "310", Size: "1.5"},
	}
	asks := []*Update{
		&Update{Price: "335.12", Size: "0.5"},
	}
	timestamp := time.Now().Unix()
	isInserted := ob.SetSnapshot(timestamp, bids, asks)
	if isInserted != true {
		t.Fail()
	}
	numBids, numAsks := ob.GetBookCount()
	if numBids != 3 || numAsks != 1 {
		t.Errorf("Num bids expected as 3, but was %d. Num asks expected was 1, but was %d", numBids, numAsks)
	}

	result, _, err := ob.SellBase(0.6)
	if err != nil {
		t.Error(err.Error())
	}
	if result != 198.6 {
		t.Errorf("Expected 198.6 but got %f", result)
	}
}

func TestUpdate(t *testing.T) {
	ob := NewOrderbookFeed("ETH-DAI")
	ob.SetSnapshot(time.Now().Unix(), []*Update{}, []*Update{})
	numBids, numAsks := ob.GetBookCount()
	if numBids != 0 || numAsks != 0 {
		t.Errorf("Num bids expected as 0, but was %d. Num asks expected was 0, but was %d", numBids, numAsks)
	}

	ob.WriteUpdate(time.Now().Unix(), []*Update{
		&Update{Price: "333.2", Size: "0.5"},
		&Update{Price: "310", Size: "1.5"},
	}, []*Update{})
	result, _, err := ob.SellBase(0.6)
	if err != nil {
		t.Error(err.Error())
	}
	if result != 197.6 {
		t.Errorf("Expected 197.6 but got %f", result)
	}
	ob.WriteUpdate(time.Now().Unix(), []*Update{
		&Update{Price: "320", Size: "0.5"},
	}, []*Update{})
	result, _, err = ob.SellBase(0.6)
	if err != nil {
		t.Error(err.Error())
	}
	if result != 198.6 {
		t.Errorf("Expected 198.6 but got %f", result)
	}

	ob.WriteUpdate(time.Now().Unix(), []*Update{
		&Update{Price: "333.2", Size: "1.5"},
	}, []*Update{})
	result, _, err = ob.SellBase(0.6)
	if err != nil {
		t.Error(err.Error())
	}
	if result != 199.92 {
		t.Errorf("Expected 199.92 but got %f", result)
	}

}

func TestBuyBase(t *testing.T) {
	ob := NewOrderbookFeed("ETH-DAI")
	bids := []*Update{
		&Update{Price: "333.2", Size: "0.5"},
		&Update{Price: "320", Size: "0.5"},
		&Update{Price: "310", Size: "1.5"},
	}
	asks := []*Update{
		&Update{Price: "335.12", Size: "0.5"},
	}
	timestamp := time.Now().Unix()
	ob.SetSnapshot(timestamp, bids, asks)
	result, _, err := ob.BuyBase(0.2)
	if err != nil {
		t.Error(err.Error())
	}
	if result != 67.024 {
		t.Errorf("Expected 198.6 but got %f", result)
	}
}

func TestBuyQuote(t *testing.T) {
	ob := NewOrderbookFeed("ETH-DAI")
	bids := []*Update{
		&Update{Price: "333.2", Size: "0.5"},
		&Update{Price: "320", Size: "0.5"},
		&Update{Price: "310", Size: "1.5"},
	}
	asks := []*Update{
		&Update{Price: "335.12", Size: "0.5"},
	}
	timestamp := time.Now().Unix()
	ob.SetSnapshot(timestamp, bids, asks)

	result, _, err := ob.BuyQuote(200)
	if err != nil {
		t.Error(err.Error())
	}
	if result != 0.604375 {
		t.Errorf("Expected 0.604375 but got %f", result)
	}
}

func TestSellQuote(t *testing.T) {
	ob := NewOrderbookFeed("ETH-DAI")
	bids := []*Update{
		&Update{Price: "333.2", Size: "0.5"},
		&Update{Price: "320", Size: "0.5"},
		&Update{Price: "310", Size: "1.5"},
	}
	asks := []*Update{
		&Update{Price: "335.12", Size: "0.5"},
	}
	timestamp := time.Now().Unix()
	ob.SetSnapshot(timestamp, bids, asks)

	result, _, err := ob.SellQuote(50)
	if err != nil {
		t.Error(err.Error())
	}
	if result != 0.14920028646455 {
		t.Errorf("Expected 0.14920028646455 but got %f", result)
	}
}

func TestErrorWillReturnIfNoSnapshotWasEverSet(t *testing.T) {
	ob := NewOrderbookFeed("ETH-DAI")
	ob.WriteUpdate(1, []*Update{
		&Update{Price: "333.2", Size: "1.5"},
	}, []*Update{})
	_, _, err1 := ob.SellBase(0.6)
	_, _, err2 := ob.BuyBase(0.6)
	if err1 == nil || err2 == nil {
		t.Error("No orderbook operations should be allowed if a snapshot was never set")
	}
}

func TestEndToEnd(t *testing.T) {
	response, err := http.Get(URL)
	defer response.Body.Close()
	if err != nil {
		panic(err)
	}

	var l2Data LevelTwoOrderbook
	decoder := json.NewDecoder(response.Body)
	decoder.Decode(&l2Data)

	ob := NewOrderbookFeed("ETH-DAI")
	bids := transformToUpdate(l2Data.Bids)
	asks := transformToUpdate(l2Data.Asks)
	ob.SetSnapshot(time.Now().Unix(), bids, asks)

	for i := 10; i < 400; i += 10 {
		quoteObtained, _, _ := ob.SellBase(float64(i))
		baseObtained, _, _ := ob.BuyQuote(quoteObtained)

		if math.Round(baseObtained) != float64(i) {
			t.Errorf("Expcted %d but got %f", i, baseObtained)
		}
	}
}

func TestTimestampUpdateIsWorking(t *testing.T) {
	ob := NewOrderbookFeed("ETH-DAI")
	ob.SetSnapshot(1, []*Update{}, []*Update{})
	bids := []*Update{
		&Update{Price: "333.2", Size: "0.5"},
		&Update{Price: "310", Size: "1.5"},
	}
	asks := []*Update{}
	timestamp := time.Now().Unix()
	isUpdatedCorrectly := ob.SetSnapshot(timestamp, bids, asks)
	if !isUpdatedCorrectly {
		t.Errorf("Update should work correctly")
	}

	bids = []*Update{
		&Update{Price: "320", Size: "0.5"},
	}
	isUpdatedCorrectly = ob.WriteUpdate(timestamp-1, bids, asks)
	if isUpdatedCorrectly {
		t.Errorf("Update should not have worked due to old timestamp")
	}
}

func TestSellQuoteWhenBookIsOld(t *testing.T) {
	ob := NewOrderbookFeed("ETH-DAI")
	bids := []*Update{
		&Update{Price: "333.2", Size: "0.5"},
		&Update{Price: "320", Size: "0.5"},
		&Update{Price: "310", Size: "1.5"},
	}
	asks := []*Update{
		&Update{Price: "335.12", Size: "0.5"},
	}
	timestamp := time.Now().Unix() - 6
	ob.SetSnapshot(timestamp, bids, asks)

	_, _, err := ob.SellQuote(50)
	if err == nil || err.Error() != "Orderbook is stale" {
		t.Error("Orderbook is stale but no error was raised")
	}

	_, _, err = ob.SellBase(50)
	if err == nil || err.Error() != "Orderbook is stale" {
		t.Error("Orderbook is stale but no error was raised")
	}
}

func TestOrderbookCleanup(t *testing.T) {
	ob := NewOrderbookFeed("ETH-DAI")
	bids := []*Update{
		&Update{Price: "333.2", Size: "0.5"},
		&Update{Price: "320", Size: "0.5"},
		&Update{Price: "310", Size: "1.5"},
	}
	asks := []*Update{
		&Update{Price: "335.12", Size: "0.5"},
	}
	ob.SetSnapshot(time.Now().Unix(), bids, asks)

	// Set price to 0
	bids = []*Update{
		&Update{Price: "333.2", Size: "0"},
	}
	ob.WriteUpdate(time.Now().Unix(), bids, asks)

	// Rows should still return 3 count
	p1, _, _ := ob.SellQuote(50)
	numBids, _ := ob.GetBookCount()
	if numBids != 3 {
		t.Errorf("Expected 3 bids, got %d", numBids)
	}

	// Clean up orderboo
	ob.CleanUpOrderbook()
	numBids, _ = ob.GetBookCount()
	if numBids != 2 {
		t.Errorf("Expected 2 bids, got %d", numBids)
	}
	p2, _, _ := ob.SellQuote(50)
	if p2 != p1 {
		t.Fail()
	}

}
