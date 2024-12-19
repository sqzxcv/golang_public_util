package websocket

import "github.com/gin-gonic/gin"

// 定义 消息回调
var BinaryMessageHandler func(client *Client, message *Req) (respData []byte, err error)

var AuthHandler func(c *gin.Context) (err error)
