package main

import (
	"fmt"
	"log"
	"time"
)

func setGridOrders(pSym string) {
	//precPrice := qtyDecimalPlaces(AssetInfo[pSym].BaseTickPrice)
	precQuoteCoin := 2 //Precision for SELL Price
	ticker := getTicker(pSym)
	cfg := Config[pSym]
	baseCoin := AssetInfo[pSym].BaseCoin
	quoteCoin := AssetInfo[pSym].QuotedCoin
	wallet := make(map[string]float64)

	avgPrice := (ticker.AskPrice + ticker.BidPrice) / 2

	fmt.Printf("%s: Current Price =  %f\n", pSym, avgPrice)

	//Check contains avgPrice for max/min price
	if avgPrice < cfg.MinPrice {
		fmt.Println(time.Now().Format("2 Jan 15:04:05"), "Current Price out of range.")
		return
	}

	//Calculate profit
	step := (cfg.MaxPrice - cfg.MinPrice) / float64(cfg.LotCount)
	// Formula: ((PriceSell/PriceBuy - 1) * 100%) - (2 * TAXRATE)
	minProfit := (((cfg.MaxPrice / (cfg.MaxPrice - step*0.5)) - 1) * 100) - (2 * TAXRATE)
	fmt.Printf("Min Profit = %.3f %%\n", minProfit)

	//Greate grid
	priceGrid := make([]float64, cfg.LotCount+1)
	priceGrid[0] = cfg.MaxPrice //First item

	for i := 1; i < len(priceGrid); i++ {
		priceGrid[i] = trnFloat(priceGrid[i-1]-step, precQuoteCoin) //Round as Quote Coin
	}
	fmt.Println("Create Grid of Orders:", priceGrid)

	//Remove nearest closed level price
	currLevel, priceGrid := getGridOrders(avgPrice, priceGrid)

	fmt.Println("Final Grid of Orders:", priceGrid)
	fmt.Printf("Current Level: %.2f\n\n", currLevel)

	//Check APIKEY, APISECRET
	if len(APIKEY) != 64 || len(APISECRET) != 64 {
		fmt.Println("Sorry, you should set the variables APIKEY and APISECRET from Binance...")		
		fmt.Println("Limit Orders don't set. Return.")
		return
	}

	//Check all SELL orders, convert to STOP_LOSS and follow to the current price
	listOrders := getOpenOrders(pSym)

	for _, v := range listOrders {
		stopPrice := trnFloat(avgPrice-step*0.5, precQuoteCoin)  //for STOP_LOSS
		flagPrice := trnFloat(currLevel-step*0.1, precQuoteCoin) //when need convert LIMIT to STOP_LOSS

		//Convert SELL LIMIT to SELL STOP_LOSS
		if v.Side == "SELL" && v.Type == "LIMIT" && v.Price == currLevel && avgPrice >= flagPrice {
			log.Printf("Cancel SELL LIMIT Order %d. Price: %f\n", v.OrderID, v.StopPrice)
			setCancelOrder(pSym, v.OrderID)
			log.Println("New STOP_LOSS_LIMIT Order. Stop Price:", stopPrice)
			setOrder(pSym, "SELL", "STOP_LOSS_LIMIT", "GTC", stopPrice, stopPrice, cfg.LotSize)
		}

		//Follow to the current price every 0.1 step
		if v.Side == "SELL" && v.Type == "STOP_LOSS_LIMIT" && v.StopPrice+step*0.1 < stopPrice {
			log.Printf("Cancel SELL STOP_LOSS_LIMIT Order %d. Stop Price: %f\n", v.OrderID, v.StopPrice)
			setCancelOrder(pSym, v.OrderID)
			log.Println("New STOP_LOSS_LIMIT Order. Stop Price:", stopPrice)
			setOrder(pSym, "SELL", "STOP_LOSS_LIMIT", "GTC", stopPrice, stopPrice, cfg.LotSize)
		}
	}

	//Set BUY/SELL LIMIT Orders
	for i := 0; i < len(priceGrid); i++ { //Loop on Grid Item
		var exist int   //if exists order
		var side string //BUY or SELL

		for _, v := range listOrders { //Loop for open list  Buy Orders
			if v.Price == priceGrid[i] {
				exist = 1
				break
			}
		}

		if exist == 0 { //If Order not exist and need a set order
			//check amount
			wallet = getWallet()

			//Item of PriceGrid higher current price and amount is enougth
			if priceGrid[i] > avgPrice && wallet[baseCoin] >= cfg.LotSize {
				side = "SELL"
			}

			//Item of PriceGrid lower current price, amount is enougth and item is not first (!= MaxPrice)
			if priceGrid[i] < avgPrice && wallet[quoteCoin] > cfg.LotSize*avgPrice && priceGrid[i] != cfg.MaxPrice {
				side = "BUY"
			}

			if side != "" {
				fmt.Printf("%s: Set %s Order. Price: %f, Size: %f\n", pSym, side, priceGrid[i], cfg.LotSize)
				setOrder(pSym, side, "LIMIT", "GTC", 0, priceGrid[i], cfg.LotSize)
			}

		}
	}

}

//Get current price and full grid order, return current level price and grid order without curred level price
func getGridOrders(pPrice float64, pGrid []float64) (float64, []float64) {
	var level float64

	for i := 1; i < len(pGrid); i++ {

		if pPrice > pGrid[i] && pPrice < pGrid[i-1] {
			//fmt.Printf("Current Price between %f and %f\n", pGrid[i-1], pGrid[i])
			st := (pGrid[i-1] - pGrid[i]) / 2
			if pPrice < pGrid[i]+st {
				level = pGrid[i] //remember current level
				pGrid = append(pGrid[:i], pGrid[i+1:]...)
			} else {
				level = pGrid[i-1] //remember current level
				pGrid = append(pGrid[:(i-1)], pGrid[(i-1)+1:]...)
			}
		}
	}
	return level, pGrid
}
