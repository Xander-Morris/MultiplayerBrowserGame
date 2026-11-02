package server

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/JGLTechnologies/gin-rate-limit"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func keyFunc(c *gin.Context) string {
	return c.ClientIP()
}

func errorHandler(c *gin.Context, info ratelimit.Info) {
	c.String(429, "Too many requests. Try again in "+time.Until(info.ResetTime).String())
}

func createEndpoints(ginServer *gin.Engine, mw gin.HandlerFunc, hubManager *HubManager) {
	ginServer.POST("/create", mw, func(c *gin.Context) {
		hub, err := hubManager.CreateHub(c.ClientIP())
		if err != nil {
			if errors.Is(err, ErrHubCreationRateLimited) {
				c.JSON(429, gin.H{"error": err.Error()})
				return
			}
			c.JSON(500, gin.H{"error": "could not create hub"})
			return
		}

		c.JSON(200, gin.H{
			"id":            hub.ID,
			"access_code":   hub.AccessCode,
			"created_at":    hub.CreatedAt,
			"websocket_url": fmt.Sprintf("/ws?hub_id=%s&access_code=%s", hub.ID, hub.AccessCode),
		})
	})
	ginServer.GET("/ws", func(c *gin.Context) {
		handleWebSocket(c, hubManager)
	})
}

func startGinServer() {
	ginServer := gin.Default()
	hubManager := NewHubManager(time.Hour, time.Minute, 5)
	defer hubManager.Close()
	store := ratelimit.RedisStore(&ratelimit.RedisOptions{
		RedisClient: redis.NewClient(&redis.Options{
			Addr: "localhost:7680",
		}),
		Rate:  time.Second,
		Limit: 5,
	})
	mw := ratelimit.RateLimiter(store, &ratelimit.Options{
		ErrorHandler: errorHandler,
		KeyFunc:      keyFunc,
	})
	createEndpoints(ginServer, mw, hubManager)
	ginServer.Run("0.0.0.0:" + os.Getenv("PORT"))
}

func StartServer() {
	loadEnv()
	startGinServer()
}

func handleWebSocket(c *gin.Context, hubManager *HubManager) {
	hubID := c.Query("hub_id")
	accessCode := c.Query("access_code")
	hub, ok := hubManager.Hub(hubID)
	if !ok || accessCode != hub.AccessCode {
		c.JSON(401, gin.H{"error": "invalid hub credentials"})
		return
	}

	options := &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Only for local development, testing right now
	}
	conn, err := websocket.Accept(c.Writer, c.Request, options)

	if err != nil {
		return
	}

	client := &Client{ID: randomID(), Conn: conn}
	hub.AddClient(client)
	defer func() {
		hubManager.RemoveClient(hubID, client.ID)
		conn.CloseNow()
	}()

	for {
		if _, _, err := conn.Read(c.Request.Context()); err != nil {
			return
		}
	}
}
