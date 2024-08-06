package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/Mauricio-Carrion/GoReactAMA/internal/api"
	"github.com/Mauricio-Carrion/GoReactAMA/internal/store/pgstore"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	pool, err := pgxpool.New(
		context.Background(), 
		fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s", 
		os.Getenv("POSTGRES_USER"), 
		os.Getenv("POSTGRES_PASSWORD"), 
		os.Getenv("POSTGRES_HOST"), 
		os.Getenv("POSTGRES_PORT"), 
		os.Getenv("POSTGRES_DB")),
	)
	
	if err != nil {
		panic(err)
	}

	defer pool.Close();

	if err := pool.Ping(context.Background()); err != nil {
		panic(err)
	}

	handler := api.NewHandler(pgstore.New(pool))

	go func () {
		if err := http.ListenAndServe(":8080", handler); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
}