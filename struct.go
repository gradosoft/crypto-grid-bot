package main

//MarketInfo is an AssetPair info from api.go. First coin - Base, second coin - Quoted
type MarketInfo struct {
	BaseCoin      string  //eg. "ETH" in case ETHUSDT
	QuotedCoin    string  //eg. "USDT" in case ETHUSDT
	BaseTickPrice float64 //tick price for ETH
	BaseStepSize  float64 //step size for ETH
}

//SymCfg use in getConfig from tools.go and fill from config.json
type SymCfg struct {
	MinPrice float64
	MaxPrice float64
	LotCount int
	LotSize  float64
}

//BookTicker fill from getTicker(), api.go
type BookTicker struct {
	Symbol   string
	BidPrice float64
	BidQty   float64
	AskPrice float64
	AskQty   float64
}

//OpenOrders fill from getOpenOrders()
type OpenOrders struct {
	Symbol    string
	OrderID   int
	Status    string //NEW, PARTIALLY_FILLED
	Side      string //BUY, SELL
	Type      string //LIMIT, STOP_LOSS, MARKET
	Price     float64
	StopPrice float64 //for STOP_LOSS
	Size      float64
}
