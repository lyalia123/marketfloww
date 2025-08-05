package domain

import "time"

type PriceUpdate struct {
	Exchange   string
	Symbol     string
	Price      float64
	ReceivedAt time.Time
	Type       string //"raw/min/max "
	AvgPrice   float64
	MinPrice   float64
	MaxPrice   float64
}

type AggregatedPrice struct {
	PairName  string
	Exchange  string
	Timestamp time.Time
	AvgPrice  float64
	MinPrice  float64
	MaxPrice  float64
}
