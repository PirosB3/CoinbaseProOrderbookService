package feed

import (
	"testing"
	"time"
)

func TestCanInitialize(t *testing.T) {
	ob := NewOrderbookFeed("ETH-DAI")
	base, quote := ob.GetProduct()
	if base != "ETH" || quote != "DAI" {
		t.Errorf("Expected ETH-DAI, instead got %s-%s", base, quote)
	}
}

func TestFailsForGetPrice(t *testing.T) {
	ob := NewOrderbookFeed("ETH-DAI")
	_, err := ob.SellBase(1.2)
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

	result, err := ob.SellBase(0.6)
	if err != nil {
		t.Error(err.Error())
	}
	if result != 198.6 {
		t.Errorf("Expected 198.6 but got %f", result)
	}
}

func TestUpdate(t *testing.T) {
	ob := NewOrderbookFeed("ETH-DAI")
	numBids, numAsks := ob.GetBookCount()
	if numBids != 0 || numAsks != 0 {
		t.Errorf("Num bids expected as 0, but was %d. Num asks expected was 0, but was %d", numBids, numAsks)
	}

	ob.WriteUpdate(1, []*Update{
		&Update{Price: "333.2", Size: "0.5"},
		&Update{Price: "310", Size: "1.5"},
	}, []*Update{})
	result, err := ob.SellBase(0.6)
	if err != nil {
		t.Error(err.Error())
	}
	if result != 197.6 {
		t.Errorf("Expected 197.6 but got %f", result)
	}
	ob.WriteUpdate(1, []*Update{
		&Update{Price: "320", Size: "0.5"},
	}, []*Update{})
	result, err = ob.SellBase(0.6)
	if err != nil {
		t.Error(err.Error())
	}
	if result != 198.6 {
		t.Errorf("Expected 198.6 but got %f", result)
	}

	ob.WriteUpdate(1, []*Update{
		&Update{Price: "333.2", Size: "1.5"},
	}, []*Update{})
	result, err = ob.SellBase(0.6)
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
	result, err := ob.BuyBase(0.2)
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

	result, err := ob.BuyQuote(200)
	if err != nil {
		t.Error(err.Error())
	}
	if result != 0.604375 {
		t.Errorf("Expected 0.604375 but got %f", result)
	}
}
