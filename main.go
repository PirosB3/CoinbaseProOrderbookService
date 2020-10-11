package main

import (
	"context"
	"net/http"
	"pirosb3/real_feed/controller"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func main() {
	ctx, _ := context.WithCancel(context.Background())

	fc2 := controller.NewFeedController(ctx, "BTC-USD")
	fc2.Start()

	fc1 := controller.NewFeedController(ctx, "ETH-USD")
	fc1.Start()

	go func() {
		for {
			time.Sleep(time.Second * 3)
			quoteAmount, err := fc1.BuyBase(50)
			if err != nil {
				log.WithField("err", err.Error()).Errorln("Error performing trade")
			}
			log.WithField("baseAmount", 50).WithField("quoteAmount", quoteAmount).Infoln("Buy base")
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
