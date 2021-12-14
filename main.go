package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

//APIKEY contain KEY from Binance
const APIKEY = "VeryLongAndVeryStrongApiKeyFromBinance"

//APISECRET contain private key from Binance
const APISECRET = "VeryLongAndVeryStrongApiSecretFromBinance"

//TAXRATE contains current Fee for each order
const TAXRATE = 0.075

//AssetInfo contains info about All Pairs
var AssetInfo map[string]*MarketInfo

//Config contains array from config.json
var Config map[string]*SymCfg


func main() {

	//Create log.txt to current directory
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile(dir+string(filepath.Separator)+"log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Error opening file:", err)
	}
	defer f.Close()

	//Multi log for file and console
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)
	//log.SetFlags(log.Ldate | log.Lmicroseconds)

	//Preload data and structures
	AssetInfo = getMarketInfo()
	Config = getConfig("config.json")

	//Start
	fmt.Println("Hello, I`m Binance Bot!")

	for {
		fmt.Println(time.Now().Format("2 Jan 15:04:05"), "Grid Binance Bot working...")
		setGridOrders("BTCUSDT")
		fmt.Println("Pause 5 sec... Press Ctrl-C for Exit...")
		time.Sleep(5 * time.Second)
	}

}
