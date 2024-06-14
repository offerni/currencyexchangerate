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

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/google/uuid"
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

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s.db", os.Getenv("DATABASE_NAME"))), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	err = db.AutoMigrate(&ExchangeRate{})
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()

	app := App{
		Db: db,
	}

	// routes
	mux.HandleFunc("/cotacao", app.CotacaoHandler)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Printf("Server Intialized on port :%s \n", port)
		err = http.ListenAndServe(fmt.Sprintf(":%s", port), mux)
		if err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	wg.Wait()
}

func (app App) CotacaoHandler(w http.ResponseWriter, r *http.Request) {
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

	var er ExchangeRateJsonResponse
	err = json.Unmarshal(result, &er)
	if err != nil {
		panic(err)
	}

	app.createExchangeRate(ctx, er)

	select {
	case <-ctx.Done():
		log.Println(ctx.Err().Error())

	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"bid": er["USDBRL"].Bid,
		}) //TODO: make it more generic so it can support different keys
	}
}

func (app App) createExchangeRate(ctx context.Context, er ExchangeRateJsonResponse) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	usdBrl := er["USDBRL"] // not really re-usable, make it generic

	err := app.Db.WithContext(ctx).Create(&ExchangeRate{
		Ask:        usdBrl.Ask,
		Bid:        usdBrl.Bid,
		Code:       usdBrl.Code,
		Codein:     usdBrl.Codein,
		CreateDate: usdBrl.CreateDate,
		High:       usdBrl.High,
		ID:         uuid.New().String(),
		Low:        usdBrl.Low,
		Name:       usdBrl.Name,
		PctChange:  usdBrl.PctChange,
		Timestamp:  usdBrl.Timestamp,
		VarBid:     usdBrl.VarBid,
	}).Error
	if err != nil {
		panic(err)
	}
}

type App struct {
	Db *gorm.DB
}

type ExchangeRateJsonResponse map[string]ExchangeRateJson

// http layer
type ExchangeRateJson struct {
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

// model/repo layer
type ExchangeRate struct {
	ID         string `gorm:"primaryKey;type:TEXT"`
	Code       string `gorm:"type:TEXT"`
	Codein     string `gorm:"type:TEXT"`
	Name       string `gorm:"type:TEXT"`
	High       string `gorm:"type:TEXT"`
	Low        string `gorm:"type:TEXT"`
	VarBid     string `gorm:"type:TEXT"`
	PctChange  string `gorm:"type:TEXT"`
	Bid        string `gorm:"type:TEXT"`
	Ask        string `gorm:"type:TEXT"`
	Timestamp  string `gorm:"type:TEXT"`
	CreateDate string `gorm:"type:TEXT"`
}
