package server

import (
	"log"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

func loadEnv() {
	_, b, _, _ := runtime.Caller(0)
	backendDir := filepath.Join(filepath.Dir(b), "..")
	envPath := filepath.Join(backendDir, ".env")
	err := godotenv.Load(envPath)

	if err != nil {
		log.Fatal("Error loading .env file")
	}
}