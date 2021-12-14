package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

//strContains check occurrence
func strContains(slice []string, item string) bool {

	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	_, ok := set[item]
	return ok
	//Example
	//s := []string{"a", "b"}
	//s1 := "a"
	//fmt.Println(strContains(s, s1))
}

//trnFloat trim float number
func trnFloat(val float64, prec int) float64 {

	rounder := math.Floor(val * math.Pow(10, float64(prec)))

	return rounder / math.Pow(10, float64(prec))
}

//qtyDecimalPlaces return qty of places after comma
func qtyDecimalPlaces(num float64) int {

	str := strconv.FormatFloat(num, 'f', -1, 64)
	if strings.Contains(str, ".") { //If exists part after "."
		parts := strings.Split(str, ".")
		return len(parts[1])
	}

	return 0
}

//getConfig read config.json
func getConfig(pFile string) map[string]*SymCfg {

	result := make(map[string]*SymCfg)

	configFile, openErr := os.Open(pFile)
	if openErr != nil {
		log.Fatal(openErr)
	}
	defer configFile.Close()

	var raw []map[string]interface{}

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&raw)

	for _, v := range raw {
		item := SymCfg{}

		item.MinPrice = v["minprice"].(float64)
		item.MaxPrice = v["maxprice"].(float64)
		item.LotCount = int(v["lotcount"].(float64))
		item.LotSize = v["lotsize"].(float64)

		result[v["symbol"].(string)] = &item
	}

	return result
}

//Signature Method
func getSignature(message string) string {
	mac := hmac.New(sha256.New, []byte(APISECRET))
	mac.Write([]byte(message))
	return fmt.Sprintf("%x", (mac.Sum(nil)))
}
