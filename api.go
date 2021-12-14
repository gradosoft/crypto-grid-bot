package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

//Connections and returns body responce
func connPublic(pURL string) ([]byte, int) { //Just GET responce without signing

	spaceClient := http.Client{
		Timeout: time.Second * 12, //Max of 12 secs
	}

	req, reqErr := http.NewRequest(http.MethodGet, pURL, nil)
	if reqErr != nil {
		log.Fatal(reqErr)
	}

	req.Header.Set("User-Agent", "Binance Bot v.0.1")

	res, resErr := spaceClient.Do(req)
	if resErr != nil {
		log.Fatal(resErr)
	}
	defer res.Body.Close()

	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Fatal(bodyErr)
	}

	if res.StatusCode != 200 {
		log.Println(pURL, res.Status)
		log.Println(string(body))
	}

	return body, res.StatusCode
}

//Binance API

func getMarketInfo() map[string]*MarketInfo { //GET /v1/mdata?cmd=marketAll

	reqURL := "https://api.binance.com/api/v1/exchangeInfo"
	body, _ := connPublic(reqURL)

	type Symbol struct {
		Sym                  string        `json:"symbol"`
		Status               string        `json:"status"`
		BaseAsset            string        `json:"baseAsset"`
		BaseAssetPrecision   int           `json:"baseAssetPrecision"`
		QuoteAsset           string        `json:"quoteAsset"`
		QuotePrecision       int           `json:"quotePrecision"`
		OrderTypes           []interface{} `json:"orderTypes"`
		IcebergAllowed       bool          `json:"icebergAllowed"`
		SpotTradingAllowed   bool          `json:"isSpotTradingAllowed"`
		MarginTradingAllowed bool          `json:"isMarginTradingAllowed"`
		Filters              []interface{} `json:"filters"`
	}

	type MarketAll struct {
		TimeZone        string        `json:"timezone"`
		ServerTime      int           `json:"servertime"`
		RateLimits      []interface{} `json:"rateLimits"`
		ExchangeFilters []interface{} `json:"exchangeFilters"`
		Symbols         []Symbol      `json:"symbols"`
	}

	raw := MarketAll{}
	jsonErr := json.Unmarshal(body, &raw)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	asset := make(map[string]*MarketInfo)

	for _, v := range raw.Symbols {

		jsonPrice, _ := json.Marshal(v.Filters[0])
		jsonSize, _ := json.Marshal(v.Filters[2])

		mapPrice := make(map[string]string)
		mapSize := make(map[string]string)

		jsonErr := json.Unmarshal(jsonPrice, &mapPrice)
		if jsonErr != nil {
			log.Fatal(jsonErr)
		}

		jsonErr = json.Unmarshal(jsonSize, &mapSize)
		if jsonErr != nil {
			log.Fatal(jsonErr)
		}

		item := MarketInfo{}
		tickPrice, _ := strconv.ParseFloat(mapPrice["tickSize"], 64)
		stepSize, _ := strconv.ParseFloat(mapSize["stepSize"], 64)

		item.BaseCoin = v.BaseAsset
		item.QuotedCoin = v.QuoteAsset
		item.BaseTickPrice = tickPrice
		item.BaseStepSize = stepSize

		asset[v.Sym] = &item

	}

	return asset
}

func getTicker(pSym string) BookTicker { //GET /api/v3/ticker/bookTicker

	reqURL := "https://api.binance.com/api/v3/ticker/bookTicker?symbol=" + pSym
	body, _ := connPublic(reqURL)

	raw := make(map[string]string)
	jsonErr := json.Unmarshal(body, &raw)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	ticker := BookTicker{}

	ticker.Symbol = raw["symbol"]
	ticker.AskPrice, _ = strconv.ParseFloat(raw["askPrice"], 64)
	ticker.AskQty, _ = strconv.ParseFloat(raw["askQty"], 64)
	ticker.BidPrice, _ = strconv.ParseFloat(raw["bidPrice"], 64)
	ticker.BidQty, _ = strconv.ParseFloat(raw["bidQty"], 64)

	return ticker
}

//setOrder pSide:"BUY/SELL", pType:"LIMIT/MARKET", pForce:"GTC/FOK". 10 orders per sec
func setOrder(pSym string, pSide string, pType string, pForce string, pStop float64, pPrice float64, pSize float64) {

	ts := int64(time.Nanosecond) * time.Now().UnixNano() / int64(time.Millisecond)
	buffer := new(bytes.Buffer)

	params := url.Values{}
	params.Set("symbol", pSym)               //e.g. BTCUSDT
	params.Set("side", pSide)                //BUY or SELL
	params.Set("type", pType)                //LIMIT, MARKET, etc...
	params.Set("newOrderRespType", "RESULT") //ACK, RESULT or FULL
	params.Set("quantity", fmt.Sprintf("%f", pSize))
	params.Set("timestamp", fmt.Sprintf("%d", ts))
	if pType == "LIMIT" {
		params.Set("timeInForce", pForce) //GTC, IOC, FOK
		params.Set("price", fmt.Sprintf("%f", pPrice))
	}
	if pType == "STOP_LOSS_LIMIT" {
		params.Set("stopPrice", fmt.Sprintf("%f", pStop))
		params.Set("price", fmt.Sprintf("%f", pPrice))
		params.Set("timeInForce", pForce) //GTC, IOC, FOK
	}
	signature := getSignature(params.Encode())
	params.Set("signature", signature)

	orderURL := "https://api.binance.com/api/v3/order"

	buffer.WriteString(params.Encode())

	client := &http.Client{}
	req, _ := http.NewRequest("POST", orderURL, buffer)
	req.Header.Set("X-MBX-APIKEY", APIKEY)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Response Error:", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Response Boby Error:", err)
	}

	if bytes.Contains(body, []byte("msg")) {
		log.Println("SetOrder Error:", string(body))
		return
	}

	//Parse response
	var raw map[string]interface{}
	jsonErr := json.Unmarshal(body, &raw)
	if jsonErr != nil {
		log.Println(jsonErr)
	}

	sym := raw["symbol"].(string)
	tp := raw["type"].(string)
	sd := raw["side"].(string)
	id := int(raw["orderId"].(float64))
	price := raw["price"].(string)
	qty := raw["origQty"].(string)
	st := raw["status"].(string)

	log.Printf("Create %s %s Order %s. ID: %d, Price: %s, Size: %s, Status: %s\n", sd, tp, sym, id, price, qty, st)
}

//setCancelOrder
func setCancelOrder(pSym string, pID int) {

	ts := int64(time.Nanosecond) * time.Now().UnixNano() / int64(time.Millisecond)
	buffer := new(bytes.Buffer)

	params := url.Values{}
	params.Set("symbol", pSym)                    //e.g. BTCUSDT
	params.Set("orderId", fmt.Sprintf("%d", pID)) //Order ID
	params.Set("timestamp", fmt.Sprintf("%d", ts))
	signature := getSignature(params.Encode())
	params.Set("signature", signature)

	orderURL := "https://api.binance.com/api/v3/order"

	buffer.WriteString(params.Encode())

	client := &http.Client{}
	req, _ := http.NewRequest("DELETE", orderURL, buffer)
	req.Header.Set("X-MBX-APIKEY", APIKEY)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Response Error:", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Response Boby Error:", err)
	}

	if bytes.Contains(body, []byte("msg")) {
		log.Println("SetOrder Error:", string(body))
		return
	}

	//Parse response
	var raw map[string]interface{}
	jsonErr := json.Unmarshal(body, &raw)
	if jsonErr != nil {
		log.Println(jsonErr)
	}

	sym := raw["symbol"].(string)
	side := raw["side"].(string)
	id := int(raw["orderId"].(float64))
	price := raw["price"].(string)
	size := raw["origQty"].(string)
	status := raw["status"].(string)

	log.Printf("Cancel %s Order %s. ID: %d, Price: %s, Size: %s, Status: %s\n", side, sym, id, price, size, status)
}

//getOpenOrders return all open orders on a symbol pSym
func getOpenOrders(pSym string) []OpenOrders {

	ts := int64(time.Nanosecond) * time.Now().UnixNano() / int64(time.Millisecond)

	params := url.Values{}
	params.Set("symbol", pSym) //e.g. BTCUSDT
	params.Set("timestamp", fmt.Sprintf("%d", ts))

	sign := getSignature(params.Encode())

	orderURL := fmt.Sprintf("https://api.binance.com/api/v3/openOrders?symbol=%s&timestamp=%d&signature=%s", pSym, ts, sign)

	client := &http.Client{}
	req, _ := http.NewRequest("GET", orderURL, nil)
	req.Header.Set("X-MBX-APIKEY", APIKEY)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Response Error:", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Response Boby Error:", err)
	}

	//Parse response
	var raw []map[string]interface{}

	jsonErr := json.Unmarshal(body, &raw)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	result := []OpenOrders{}

	for _, v := range raw {
		item := OpenOrders{}

		item.Symbol = v["symbol"].(string)
		item.OrderID = int(v["orderId"].(float64))
		item.Status = v["status"].(string)
		item.Side = v["side"].(string)
		item.Type = v["type"].(string)
		item.Price, _ = strconv.ParseFloat(v["price"].(string), 64)
		item.StopPrice, _ = strconv.ParseFloat(v["stopPrice"].(string), 64)
		item.Size, _ = strconv.ParseFloat(v["origQty"].(string), 64)

		result = append(result, item)
	}

	return result
}

//getWallet return map["BTC"]0.0056432
func getWallet() map[string]float64 {

	wallet := make(map[string]float64)

	ts := int64(time.Nanosecond) * time.Now().UnixNano() / int64(time.Millisecond)

	params := url.Values{}
	params.Set("timestamp", fmt.Sprintf("%d", ts))

	sign := getSignature(params.Encode())

	orderURL := fmt.Sprintf("https://api.binance.com/api/v3/account?timestamp=%d&signature=%s", ts, sign)

	client := &http.Client{}
	req, _ := http.NewRequest("GET", orderURL, nil)
	req.Header.Set("X-MBX-APIKEY", APIKEY)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Response Error:", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Response Boby Error:", err)
	}

	type AccountInfo struct {
		Balances []map[string]string `json:"balances"`
	}

	raw := AccountInfo{}
	jsonErr := json.Unmarshal(body, &raw)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	for _, v := range raw.Balances {
		size, _ := strconv.ParseFloat(v["free"], 64)
		if size > 0 {
			wallet[v["asset"]] = size
		}
	}

	return wallet
}
