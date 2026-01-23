package websocket

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sqzxcv/glog"
)

// ClientManager is a websocket manager
type ClientManager struct {
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	clientPool sync.Pool
	// 接受客户端request, 并返回Response
	ReqResponseMessageHandler ReqResponseMessageHandler
	PushMessageHandler        PushMessageHandler

	// NewClientRegisterHandler is invoked when a new client successfully registers with the ClientManager.
	NewClientRegisterHandler func(client *Client)
	// ClientWillUnRegisterHandler is invoked when a client will be unregistered from the ClientManager.
	ClientWillUnRegisterHandler func(client *Client)
	// websocket链接成功后,第一条消息是授权消息, 调用它开始鉴权
	FirstMessageAuthHandler AuthHandler
}

func NewClientManager(registerHandler func(client *Client), unRegisterHandler func(client *Client), firstMsgAuthHandler AuthHandler, reqResponseMsgHandler ReqResponseMessageHandler, pushMsgHandler PushMessageHandler) *ClientManager {
	// Manager define a ws server manager
	var manager = &ClientManager{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte, 256),
		Register:   make(chan *Client, 256),
		Unregister: make(chan *Client, 256),
		clientPool: sync.Pool{
			New: func() any {
				return new(Client)
			},
		},
		ReqResponseMessageHandler:   reqResponseMsgHandler,
		PushMessageHandler:          pushMsgHandler,
		NewClientRegisterHandler:    registerHandler,
		ClientWillUnRegisterHandler: unRegisterHandler,
		FirstMessageAuthHandler:     firstMsgAuthHandler,
	}
	return manager
}

// Message is an object for websocket message which is mapped to json type
//type Message struct {
//	Sender     string `json:"sender,omitempty"`
//	Recipient  string `json:"recipient,omitempty"`
//	Content    string `json:"content,omitempty"`
//
//}

// Start is to start a ws server
func (manager *ClientManager) Start() {
	defer glog.FInfo("ws server exit")
	for {
		select {
		case client := <-manager.Register:
			if manager.NewClientRegisterHandler != nil {
				// 不能做耗时操作, 否则会阻塞事件循环
				manager.NewClientRegisterHandler(client)
			}
			manager.Clients[client] = true

		case client := <-manager.Unregister:
			glog.FInfo("unregister client:%s", client.ID)
			if _, ok := manager.Clients[client]; ok {
				delete(manager.Clients, client)
			}
			for c, _ := range manager.Clients {
				glog.FInfo("existed client:%s", c.ID)
			}
			if manager.ClientWillUnRegisterHandler != nil {
				// 不能做耗时操作, 否则会阻塞事件循环
				manager.ClientWillUnRegisterHandler(client)
			}
			client.close(nil)
		case message := <-manager.Broadcast:
			//for client := range manager.Clients {
			//	client.PushMessage(message)
			//}
			for c := range manager.Clients {
				select {
				case c.sendCh <- message:
				default:
					// 队列满：丢弃
				}
			}

		}
	}
}

func (manager *ClientManager) UnRegister(client *Client) {
	manager.Unregister <- client
}

// WsHandler is a websocket handler
// 将普通http请求升级为websocket协议
func (manager *ClientManager) WsHandler(c *gin.Context) {

	// change the reqest to websocket model
	conn, err := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		glog.FError("Failed to upgrade request to WebSocket: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "WebSocket upgrade failed"})
		return
	}

	var sessionId string
	if manager.FirstMessageAuthHandler != nil {
		sessionId, err = manager.FirstMessageAuthHandler(c, conn)
		if err != nil {
			//http.NotFound(c.Writer, c.Request)
			//http.Error(c.Writer, "Auth failed", http.StatusUnauthorized)
			//closeCode := http.StatusUnauthorized // 1000: 正常关闭

			//closeReason := "http.StatusUnauthorized"
			//err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, closeReason))
			//if err != nil {
			//	fmt.Println("发送关闭帧失败:", err)
			//}
			//c.AbortWithStatus(http.StatusUnauthorized)
			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(1008, err.Error()),
				time.Now().Add(time.Second),
			)
			return
		}
	}

	client := manager.clientPool.Get().(*Client)
	client.ResetClient(c, conn, manager, sessionId)

	manager.Register <- client

	go client.readMessage()
	go client.writePump()
}

//WSBroadcast 向客户端广播json消息.
func (manager *ClientManager) WSBroadcast(message []byte) {

	//if len(Manager.Clients) == 0 {
	//	glog.Info("WSBroadcast no client")
	//	return
	//}
	//glog.Info("WSBroadcast hasClient:", len(manager.Clients))
	msg := Resp{
		ReqIdentifier: WSPushMsg,
		Data:          string(message),
	}
	data := []byte(structToJsonString(msg))
	manager.Broadcast <- data
}

// 心跳机制: 服务器每隔一段时间广播一条心跳包. 客户端检测心跳包,如果长时间没收到心跳包, 客户端则主动端口链接重连
//func heartBeat() {
//
//	WSBroadcast("beat", "beat")
//	time.AfterFunc(time.Second*20, func() {
//		heartBeat()
//	})
//}
