package server

import (
	"fmt"
	"log"
	"os"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func StartServer() {
	err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }
	
	ginServer := gin.Default()
	ginServer.POST("/create", func(c *gin.Context) {
		fmt.Println(c.Params)
	})
	ginServer.GET("/ws", handleWebSocket)
	ginServer.Run(os.Getenv("SERVER_ADDRESS"))
}

func handleWebSocket(c *gin.Context) {
	options := &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Only for local development, testing right now
	}
	conn, err := websocket.Accept(c.Writer, c.Request, options)

	if err != nil {
		return
	}

	defer conn.CloseNow()
}
