package types

type Product struct {
	ProductType     int    `json:"productType"`
	ProductSwitch   bool   `json:"productSwitch"`
	Amount          string `json:"amount"`
	PrepaidAmount   string `json:"prepaidAmount"`
	SubscribePeriod int    `json:"subscribePeriod"`
	VipDays         int    `json:"vipDays"`
	TopText         string `json:"topText"`
	BottomText      string `json:"bottomText"`
}
