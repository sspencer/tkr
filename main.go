package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
)

type Config struct {
	ApiKey    string   `toml:"api_key"`
	Crypto    []string `toml:"crypto"`
	StockUrl  string   `toml:"stock_url"`
	CryptoUrl string   `toml:"crypto_url"`
}

type StockQuote struct {
	GlobalQuote struct {
		Symbol           string `json:"01. symbol"`
		Open             string `json:"02. open"`
		High             string `json:"03. high"`
		Low              string `json:"04. low"`
		Price            string `json:"05. price"`
		Volume           string `json:"06. volume"`
		LatestTradingDay string `json:"07. latest trading day"`
		PreviousClose    string `json:"08. previous close"`
		Change           string `json:"09. change"`
		ChangePercent    string `json:"10. change percent"`
	} `json:"Global Quote"`
}

type CryptoQuote struct {
	RealtimeCurrencyExchangeRate struct {
		FromCurrencyCode string `json:"1. From_Currency Code"`
		FromCurrencyName string `json:"2. From_Currency Name"`
		ToCurrencyCode   string `json:"3. To_Currency Code"`
		ToCurrencyName   string `json:"4. To_Currency Name"`
		ExchangeRate     string `json:"5. Exchange Rate"`
		LastRefreshed    string `json:"6. Last Refreshed"`
		TimeZone         string `json:"7. Time Zone"`
		BidPrice         string `json:"8. Bid Price"`
		AskPrice         string `json:"9. Ask Price"`
	} `json:"Realtime Currency Exchange Rate"`
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func main() {
	alfredPtr := flag.Bool("a", false, "convert output for use with Alfred workflow")
	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		fmt.Println("Fetch stock or crypto quote")
		fmt.Printf("Usage: %s [-a] symbol\n", os.Args[0])
		os.Exit(1)
	}

	profile, err := user.Current()
	if err != nil {
		exit(err)
	}

	var config Config
	_, err = toml.DecodeFile(path.Join(profile.HomeDir, ".tkr.toml"), &config)
	if err != nil {
		exit(err)
	}

	symbol := strings.ToUpper(args[0])
	isCrypto := containsCrypto(config, symbol)
	uri := formatUri(config, symbol, isCrypto)

	resp, err := http.Get(uri)
	if err != nil {
		exit(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		exit(err)
	}

	if *alfredPtr == false {
		fmt.Println(string(body))
		os.Exit(0)
	}

	if isCrypto {
		cryptoQuote(body)
	} else {
		stockQuote(body)
	}
}

func stockQuote(body []byte) {
	quote := StockQuote{}
	if err := json.Unmarshal(body, &quote); err != nil {
		exit(err)
	}

	symbol := quote.GlobalQuote.Symbol
	price := atof(quote.GlobalQuote.Price)
	change := atof(quote.GlobalQuote.Change)
	high := atof(quote.GlobalQuote.High)
	low := atof(quote.GlobalQuote.Low)
	emoji := "📈"

	if change < 0.0 {
		emoji = "📉"
	}

	title := fmt.Sprintf("%s: %0.2f | Change %0.2f %s", symbol, price, change, emoji)
	subtitle := fmt.Sprintf("High: %0.2f | Low: %0.2f", high, low)
	fmt.Printf("<items><item uuid=%q arg=%q><title>%s</title><subtitle>%s</subtitle><icon>icon.png</icon></item></items>", "tkr", symbol, title, subtitle)
}

func cryptoQuote(body []byte) {
	quote := CryptoQuote{}
	if err := json.Unmarshal(body, &quote); err != nil {
		exit(err)
	}

	symbol := quote.RealtimeCurrencyExchangeRate.FromCurrencyName
	price := atof(quote.RealtimeCurrencyExchangeRate.ExchangeRate)

	title := fmt.Sprintf("%s: %0.2f", symbol, price)
	fmt.Printf("<items><item uuid=%q arg=%q><title>%s</title><icon>icon.png</icon></item></items>", "tkr", symbol, title)
}

func atof(s string) float64 {
	if n, err := strconv.ParseFloat(s, 64); err != nil {
		return 0.0
	} else {
		return n
	}
}

func formatUri(config Config, symbol string, isCrypto bool) string {
	const apikeyVar = "{api_key}"
	const symbolVar = "{symbol}"

	uri := config.StockUrl
	if isCrypto {
		uri = config.CryptoUrl
	}

	uri = strings.ReplaceAll(uri, apikeyVar, config.ApiKey)
	return strings.ReplaceAll(uri, symbolVar, symbol)
}

func containsCrypto(config Config, symbol string) bool {
	for _, s := range config.Crypto {
		if symbol == strings.ToUpper(s) {
			return true
		}
	}

	return false
}
