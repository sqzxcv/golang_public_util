package websocket

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sqzxcv/glog"
	"net/http"
	"sync"
)

// ClientManager is a websocket manager
type ClientManager struct {
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	clientPool sync.Pool
}

// Message is an object for websocket message which is mapped to json type
//type Message struct {
//	Sender     string `json:"sender,omitempty"`
//	Recipient  string `json:"recipient,omitempty"`
//	Content    string `json:"content,omitempty"`
//
//}

// Manager define a ws server manager
var Manager = &ClientManager{
	Broadcast:  make(chan []byte),
	Register:   make(chan *Client),
	Unregister: make(chan *Client),
	Clients:    make(map[*Client]bool),
	clientPool: sync.Pool{
		New: func() any {
			return new(Client)
		},
	},
}

// Start is to start a ws server
func (manager *ClientManager) Start() {
	for {
		select {
		case client := <-manager.Register:
			manager.Clients[client] = true

		case client := <-manager.Unregister:
			if _, ok := manager.Clients[client]; ok {
				delete(manager.Clients, client)
			}
			client.close()
		case message := <-manager.Broadcast:
			for client := range manager.Clients {
				client.PushMessage(message)
			}
		}
	}
}

func (manager ClientManager) UnRegister(client *Client) {
	manager.Unregister <- client
}

// WsHandler is a websocket handler
// 将普通http请求升级为websocket协议
func WsHandler(c *gin.Context) {
	// change the reqest to websocket model
	conn, err := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}

	if AuthHandler != nil {
		err := AuthHandler(c)
		if err != nil {
			//http.NotFound(c.Writer, c.Request)
			//http.Error(c.Writer, "Auth failed", http.StatusUnauthorized)
			closeCode := http.StatusUnauthorized // 1000: 正常关闭
			closeReason := "http.StatusUnauthorized"
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, closeReason))
			if err != nil {
				fmt.Println("发送关闭帧失败:", err)
			}
			return
		}
	}

	client := Manager.clientPool.Get().(*Client)
	client.ResetClient(c, conn, Manager)

	Manager.Register <- client

	go client.readMessage()
}

//WSBroadcast 向客户端广播json消息.
func WSBroadcast(message []byte) {

	//if len(Manager.Clients) == 0 {
	//	glog.Info("WSBroadcast no client")
	//	return
	//}
	glog.Info("WSBroadcast hasClient:", len(Manager.Clients))
	Manager.Broadcast <- message
}

// 心跳机制: 服务器每隔一段时间广播一条心跳包. 客户端检测心跳包,如果长时间没收到心跳包, 客户端则主动端口链接重连
//func heartBeat() {
//
//	WSBroadcast("beat", "beat")
//	time.AfterFunc(time.Second*20, func() {
//		heartBeat()
//	})
//}
