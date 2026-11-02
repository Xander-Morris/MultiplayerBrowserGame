package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func postWithRetry(url string, body []byte) (*http.Response, error) {
	var lastErr error
	for i := 0; i < 50; i++ {
		resp, err := http.Post(url, "application/json", bytes.NewReader(body))
		if err == nil {
			return resp, nil
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}
	return nil, lastErr
}

func TestStartServer(t *testing.T) {
	go func() {
		StartServer()
	}()
	loadEnv()
	serverAddr := os.Getenv("BASE_ADDRESS") + ":" + os.Getenv("PORT")
	data := map[string]string{
		"name":        "Xander's Server",
		"max_players": "5",
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	resp, err := postWithRetry(serverAddr+"/create", jsonBytes)
	if err != nil {
		t.Fatalf("Error executing request: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	bodyString := string(bodyBytes)
	fmt.Println(bodyString)
	// TODO: Have to add actual assertions 
}
