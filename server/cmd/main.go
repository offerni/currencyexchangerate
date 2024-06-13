package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	initializeServer()
}

func initializeServer() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("err loading: %v", err)
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// routes
	mux.HandleFunc("/cotacao", cotacaoHandler)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Printf("Server Intialized on port :%s \n", port)
		http.ListenAndServe(fmt.Sprintf(":%s", port), mux)
	}()

	wg.Wait()
}

func cotacaoHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
	defer cancel()

	defer fmt.Println("Request Finished!")

	apiBaseUrl := os.Getenv("API_BASE_URL")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/USD-BRL", apiBaseUrl), nil)
	if err != nil {
		panic(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	result, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var er ExchangeRateResponse
	err = json.Unmarshal(result, &er)
	if err != nil {
		panic(err)
	}

	select {
	case <-ctx.Done():
		log.Println("FODEU" + ctx.Err().Error())

	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(er["USDBRL"].Bid) //TODO: make it more generic so it can support different keys
	}

}

type ExchangeRateResponse map[string]ExchangeRate

type ExchangeRate struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}
