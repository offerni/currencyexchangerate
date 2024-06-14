package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	defer sqlDB.Close()

	if err := db.AutoMigrate(&ExchangeRate{}); err != nil {
		panic(err)
	}

	mux := http.NewServeMux()

	app := App{
		Db: db,
	}

	// routes
	mux.HandleFunc("/cotacao", app.CotacaoHandler)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}

	go func() {
		fmt.Printf("Server Intialized on port :%s \n", port)
		err = srv.ListenAndServe()
		if err != nil && http.ErrServerClosed != err {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("Shutting Down Server...")

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Could not shutdown server %v\n", err)
	}
	fmt.Println("Server Stopped")

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
	if err := json.Unmarshal(result, &er); err != nil {
		panic(err)
	}

	if err := app.createExchangeRate(ctx, er); err != nil {
		panic(err)
	}

	if err := createCurrentExchangeRateFile(er); err != nil {
		panic(err)
	}

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

func (app App) createExchangeRate(ctx context.Context, er ExchangeRateJsonResponse) error {
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
		return err
	}

	return nil
}

func createCurrentExchangeRateFile(er ExchangeRateJsonResponse) error {
	f, err := os.Create("cotacao.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write([]byte(fmt.Sprintf("Dollar: %s", er["USDBRL"].Bid)))
	if err != nil {
		return err
	}

	return nil
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
