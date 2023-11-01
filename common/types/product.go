package types

type Product struct {
	Id              int     `json:"id"`
	ProductId       string  `json:"product_id"`
	ProductName     string  `json:"product_name"`
	Amount          float64 `json:"amount"`
	ProductType     int     `json:"product_type"`
	VipDay          int     `json:"vip_day"`
	TopText         string  `json:"top_text"`
	BottomText      string  `json:"bottom_text"`
	PrepaidAmount   float64 `json:"prepaid_amount"`
	TryVipDay       int     `json:"try_vip_day"`
	SubscribePeriod int     `json:"subscribe_period"`
}
