package controller

import (
	"context"
	"fmt"

	"pirosb3/real_feed/rpc"
)

type OrderbookGrpcController struct {
	rpc.UnimplementedOrderbookServiceServer
	feedController *FeedController
	product        string
}

func NewOrderbookGrpcController(feedController *FeedController, product string) *OrderbookGrpcController {
	return &OrderbookGrpcController{
		feedController: feedController,
		product:        product,
	}
}

func (ob *OrderbookGrpcController) handleResponse(response float64, lastUpdated int64, err error, productRequested string) (*rpc.PricingResponse, error) {
	if ob.product != productRequested {
		return &rpc.PricingResponse{
			Product: ob.product,
			Error:   fmt.Sprintf("Requested quote for feed '%s', but service is serving feed '%s'", productRequested, ob.product),
		}, nil
	}
	if err != nil {
		return &rpc.PricingResponse{
			Product: ob.product,
			Error:   err.Error(),
		}, nil
	}
	return &rpc.PricingResponse{
		Product:     ob.product,
		LastUpdated: lastUpdated,
		OutAmount:   float32(response),
	}, nil
}

func (ob OrderbookGrpcController) BuyBase(ctx context.Context, in *rpc.PricingRequest) (*rpc.PricingResponse, error) {
	response, lastUpdated, err := ob.feedController.BuyBase(float64(in.GetInAmount()))
	return ob.handleResponse(response, lastUpdated, err, in.GetProduct())
}

func (ob OrderbookGrpcController) BuyQuote(ctx context.Context, in *rpc.PricingRequest) (*rpc.PricingResponse, error) {
	response, lastUpdated, err := ob.feedController.BuyQuote(float64(in.GetInAmount()))
	return ob.handleResponse(response, lastUpdated, err, in.GetProduct())
}

func (ob OrderbookGrpcController) SellBase(ctx context.Context, in *rpc.PricingRequest) (*rpc.PricingResponse, error) {
	response, lastUpdated, err := ob.feedController.SellBase(float64(in.GetInAmount()))
	return ob.handleResponse(response, lastUpdated, err, in.GetProduct())
}

func (ob OrderbookGrpcController) SellQuote(ctx context.Context, in *rpc.PricingRequest) (*rpc.PricingResponse, error) {
	response, lastUpdated, err := ob.feedController.SellQuote(float64(in.GetInAmount()))
	return ob.handleResponse(response, lastUpdated, err, in.GetProduct())
}

// func (ob OrderbookGrpcController) mustEmbedUnimplementedOrderbookServiceServer() {}
