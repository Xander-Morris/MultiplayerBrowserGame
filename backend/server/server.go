package server

import (
	"os"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

func StartServer() {
	loadEnv()
	ginServer := gin.Default()
	ginServer.POST("/create", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "received",
		})
	})
	ginServer.GET("/ws", handleWebSocket)
	ginServer.Run(os.Getenv("GIN_ADDRESS"))
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
