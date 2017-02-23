package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/urfave/cli"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
)

type H map[string]interface{}

var upgrader = websocket.Upgrader{}
var stocks map[string]interface{}
var stockMutex sync.Mutex

var connectionPool = struct {
	sync.RWMutex
	connections map[*websocket.Conn]struct{}
}{
	connections: make(map[*websocket.Conn]struct{}),
}

type Stock struct {
	Code string
}

func mainHandler(c echo.Context) error {
	return c.File("templates/base.html")
}

func sendMessageToAllPool(msg WSMsg) error {
	connectionPool.RLock()
	defer connectionPool.RUnlock()
	for connection := range connectionPool.connections {
		if err := connection.WriteJSON(msg); err != nil {
			return err
		}
	}
	return nil
}

type WSMsg struct {
	Action string
	Data   interface{}
}

func wsHandler(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	connectionPool.Lock()
	connectionPool.connections[ws] = struct{}{}
	defer func(connection *websocket.Conn) {
		connectionPool.Lock()
		delete(connectionPool.connections, connection)
		connectionPool.Unlock()
	}(ws)
	connectionPool.Unlock()

	msg := WSMsg{"init", stocks}
	if err := ws.WriteJSON(msg); err != nil {
		panic(err)
	}

	for {
		// Read
		var msg WSMsg
		err := ws.ReadJSON(&msg)
		if err != nil {
			if websocket.IsCloseError(err, 1001) {
				panic(err)
			}
			continue
		}
		if msg.Action == "add" {
			code := strings.ToUpper(msg.Data.(string))
			if _, ok := stocks[code]; ok {
				continue
			}
			resp, _ := http.Get(fmt.Sprintf("http://watchstocks.herokuapp.com/api/stocks/%s", code))
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			var test map[string]interface{}
			json.Unmarshal(body, &test)
			if _, ok := test["quandl_error"]; ok {
				continue
			}
			stockMutex.Lock()
			stocks[code] = test
			stockMutex.Unlock()
			sendMessageToAllPool(WSMsg{"add", test})
		} else if msg.Action == "del" {
			code := strings.ToUpper(msg.Data.(string))
			if _, ok := stocks[code]; !ok {
				continue
			}
			stockMutex.Lock()
			delete(stocks, code)
			stockMutex.Unlock()
			sendMessageToAllPool(WSMsg{"del", code})
		}
	}
	return nil
}

func start(c *cli.Context) error {
	stocks = make(map[string]interface{})
	port := c.Int("port")
	e := echo.New()
	e.Static("/public", "public")
	e.Debug = true
	e.Logger.SetLevel(log.INFO)
	e.Use(middleware.Recover())
	e.GET("/", mainHandler)
	e.GET("/ws", wsHandler)
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", port)))
	return nil
}

func main() {
	app := cli.NewApp()
	app.Author = "Alain Gilbert"
	app.Email = "alain.gilbert.15@gmail.com"
	app.Name = "FCC pinterest app"
	app.Usage = "FCC pinterest app"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:   "port",
			Value:  3001,
			Usage:  "Webserver port",
			EnvVar: "PORT",
		},
	}
	app.Action = start
	app.Run(os.Args)
}
