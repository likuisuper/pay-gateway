package types

type Product struct {
	ProductId       string  `json:"product_id"`
	ProductName     string  `json:"product_name"`
	ProductType     int     `json:"product_type"`
	Amount          float64 `json:"amount"`
	PrepaidAmount   string  `json:"prepaid_amount"`
	SubscribePeriod int     `json:"subscribe_period"`
	VipDays         int     `json:"vip_day"`
	TopText         string  `json:"top_text"`
	BottomText      string  `json:"bottom_text"`
	TryVipDay       int     `json:"try_vip_day"`
}
