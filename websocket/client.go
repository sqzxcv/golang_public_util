// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package websocket

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
	"github.com/gofrs/uuid"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sqzxcv/glog"
)

var (
	ErrConnClosed                = errors.New("conn has closed")
	ErrNotSupportMessageProtocol = errors.New("not support message protocol")
	ErrClientClosed              = errors.New("client actively close the connection")
	ErrPanic                     = errors.New("panic error")
)

const (
	// MessageText is for UTF-8 encoded text messages like JSON.
	MessageText = iota + 1
	// MessageBinary is for binary messages like protobufs.
	MessageBinary
	// CloseMessage denotes a close control message. The optional message
	// payload contains a numeric code and text. Use the FormatCloseMessage
	// function to format a close message payload.
	CloseMessage = 8

	// PingMessage denotes a ping control message. The optional message payload
	// is UTF-8 encoded text.
	PingMessage = 9

	// PongMessage denotes a pong control message. The optional message payload
	// is UTF-8 encoded text.
	PongMessage = 10
)

type PingPongHandler func(string) error

type Client struct {
	w          *sync.Mutex
	Conn       *websocket.Conn
	PlatformID int `json:"platformID"`
	//IsCompress   bool   `json:"isCompress"`
	UserID        string `json:"userID"`
	Ctx           *gin.Context
	clientManager *ClientManager
	closed        atomic.Bool
	closedErr     error
	token         string
	ID            string
	sendCh        chan []byte // 带缓冲, 发送队列

}

// ResetClient updates the client's state with new connection and context information.
func (c *Client) ResetClient(ctx *gin.Context, conn *websocket.Conn, manager *ClientManager, sessionID string) {
	c.w = new(sync.Mutex)
	c.Conn = conn
	//c.PlatformID = stringutil.StringToInt(ctx.GetPlatformID())
	//c.IsCompress = ctx.GetCompression()
	//c.IsBackground = ctx.GetBackground()
	//c.UserID = ctx.Get("userID").(string)
	c.UserID = ctx.GetString("userID")
	c.Ctx = ctx
	c.clientManager = manager
	c.closed.Store(false)
	c.closedErr = nil
	c.sendCh = make(chan []byte, 1024)
	c.token = ""
	// ID 改为uuid
	id := ""
	if sessionID == "" {
		u4, err2 := uuid.NewV4()
		if err2 != nil {
			id = ctx.GetString("userID") + "_" + fmt.Sprintf("%d", time.Now().UnixNano())
		} else {
			id = u4.String()
		}
	} else {
		id = sessionID
	}
	c.ID = id
	//c.token = ctx.GetToken()
}

func pongWaitTime() time.Time {
	return time.Now().Add(pongWait)
}

func (c *Client) pingHandler(_ string) error {
	if err := c.Conn.SetReadDeadline(pongWaitTime()); err != nil {
		return err
	}

	return c.writePongMsg()
}

// readMessage continuously reads messages from the connection.
func (c *Client) readMessage() {
	defer func() {
		if err := recover(); err != nil {
			c.closedErr = ErrPanic
			glog.Error("socket have panic err:", err, string(debug.Stack()))
		}

		c.close(c.closedErr)
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(pongWaitTime())
	c.Conn.SetPingHandler(c.pingHandler)

	for {
		glog.Info("readMessage")
		messageType, message, returnErr := c.Conn.ReadMessage()
		if returnErr != nil {
			glog.Warn("readMessage:", returnErr, "messageType:", messageType)
			c.closedErr = returnErr
			return
		}

		//glog.Debug("readMessage-", "messageType:", messageType)
		if c.closed.Load() {
			// The scenario where the connection has just been closed, but the coroutine has not exited
			c.closedErr = ErrConnClosed
			return
		}

		switch messageType {
		case MessageBinary:
			_ = c.Conn.SetReadDeadline(pongWaitTime())
			parseDataErr := c.handleMessage(message)
			if parseDataErr != nil {
				c.closedErr = parseDataErr
				return
			}
		case MessageText:
			//c.closedErr = ErrNotSupportMessageProtocol
			//return
			_ = c.Conn.SetReadDeadline(pongWaitTime())
			parseDataErr := c.handleMessage(message)
			if parseDataErr != nil {
				c.closedErr = parseDataErr
				return
			}

		case PingMessage:
			err := c.writePongMsg()
			if err != nil {
				glog.Warn("writePongMsg:", err)
			}

		case CloseMessage:
			c.closedErr = ErrClientClosed
			return
		default:
		}
	}
}

// handleMessage processes a single message received by the client.
func (c *Client) handleMessage(message []byte) error {
	var binaryReq = getReq()
	defer freeReq(binaryReq)

	// message转化成Req
	err := binaryReq.DecodeFromMsg(message)
	if err != nil {
		glog.Error("handleBinaryMessage DecodeFromMsg error:", err)
		return err
	}

	//if binaryReq.SendID != c.UserID {
	//	return fmt.Errorf("exception conn userID not same to req userID", "binaryReq", binaryReq.String())
	//}

	glog.Debug("gateway req message", "req", binaryReq.String())

	var (
		resp       []byte
		messageErr error
	)
	if binaryReq.ReqIdentifier == WsHeartbeat {
		resp, messageErr = c.handleHeartBeat(c.Ctx, binaryReq)
	} else if binaryReq.ReqIdentifier == WSPushMsg {
		if c.clientManager.PushMessageHandler != nil {
			// 如果error 不为nil, 则将关闭socket
			return c.clientManager.PushMessageHandler(c, binaryReq)
		}
		return nil
	} else if c.clientManager.ReqResponseMessageHandler != nil {
		resp, messageErr = c.clientManager.ReqResponseMessageHandler(c, binaryReq)
	}

	return c.replyMessage(binaryReq, messageErr, resp)
}

func (c *Client) handleHeartBeat(ctx context.Context, req *Req) ([]byte, error) {
	return []byte(req.Data), nil
}

func (c *Client) close(err error) {
	if c.closed.Load() {
		return
	}

	c.w.Lock()
	defer c.w.Unlock()

	if c.closed.Load() {
		return
	}
	c.closed.Store(true)

	// 根据错误选择关闭码与理由
	code := websocket.CloseNormalClosure // 1000
	reason := ""
	if err != nil {
		switch {
		case errors.Is(err, ErrClientClosed):
			code = websocket.CloseNormalClosure
			reason = err.Error()
		case errors.Is(err, ErrPanic):
			code = websocket.CloseInternalServerErr // 1011
			reason = err.Error()
		default:
			code = websocket.CloseInternalServerErr
			reason = err.Error()
		}
	}

	// 尝试发送 Close 控制帧（带 code/reason）
	_ = c.Conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(code, reason),
		time.Now().Add(time.Second),
	)

	_ = c.Conn.Close()
	c.clientManager.UnRegister(c)
}

func (c *Client) replyMessage(binaryReq *Req, err error, resp []byte) error {
	mReply := Resp{
		ReqIdentifier: binaryReq.ReqIdentifier,
		MsgIncr:       binaryReq.MsgIncr,
		OperationID:   binaryReq.OperationID,
		//ErrCode:       errResp.ErrCode,
		//ErrMsg: err.Error(),
		//Data:   string(resp),
	}
	if err != nil {
		mReply.ErrCode = 1
		mReply.ErrMsg = err.Error()
	} else {
		mReply.Data = string(resp)
	}
	glog.Debug("gateway reply message", "resp", mReply.String())
	err = c.writeBinaryMsg(mReply)
	if err != nil {
		glog.Warn("wireBinaryMsg replyMessage", err, "resp", mReply.String())
	}

	if binaryReq.ReqIdentifier == WsLogoutMsg {
		return fmt.Errorf("user logout", "operationID", binaryReq.OperationID)
	}
	return nil
}

func (c *Client) PushMessage(msgData []byte) error {

	//glog.Debug("PushMessage")
	resp := Resp{
		ReqIdentifier: WSPushMsg,
		Data:          string(msgData),
	}
	return c.writeBinaryMsg(resp)
}

func (c *Client) writePump() {
	for msg := range c.sendCh {
		// 与 writeBinaryMsg/writePongMsg 共用同一把写锁,确保同一连接同一时刻只有一个写者
		// (gorilla/websocket 要求),否则推送帧与心跳/pong 帧交错会损坏帧、导致前端心跳误判重连
		c.w.Lock()
		c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		err := c.Conn.WriteMessage(websocket.TextMessage, msg)
		c.w.Unlock()
		if err != nil {
			// 失败则触发注销
			c.clientManager.Unregister <- c
			return
		}
	}
}

func (c *Client) writeBinaryMsg(resp Resp) error {
	if c.closed.Load() {
		return nil
	}

	encodedBuf := resp.String()

	c.w.Lock()
	defer c.w.Unlock()

	err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err != nil {
		return err
	}

	return c.Conn.WriteMessage(MessageBinary, []byte(encodedBuf))
}

func (c *Client) writePongMsg() error {
	if c.closed.Load() {
		return nil
	}

	c.w.Lock()
	defer c.w.Unlock()

	err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err != nil {
		return err
	}

	return c.Conn.WriteMessage(PongMessage, nil)
}
