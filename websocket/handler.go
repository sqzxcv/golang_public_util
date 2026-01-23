package websocket

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// 定义 消息回调
type ReqResponseMessageHandler func(client *Client, message *Req) (respData []byte, err error)

// PushMessageHandler 如果Error为空则不会返回任何数据给client, 如果Error!= nil, 则关闭websocket
type PushMessageHandler func(client *Client, message *Req) (err error)

//type NewClientConnectSuccessHandler func(client *Client)

// 如果返回会话sessionID不为空, 这个将作为后面client的ID, 这个目前主要在外部调用时使用
type AuthHandler func(c *gin.Context, conn *websocket.Conn) (sessionId string, err error)
