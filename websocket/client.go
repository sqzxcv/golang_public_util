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
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sqzxcv/glog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
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
	conn       *websocket.Conn
	PlatformID int `json:"platformID"`
	//IsCompress   bool   `json:"isCompress"`
	UserID        string `json:"userID"`
	ctx           *gin.Context
	clientManager *ClientManager
	closed        atomic.Bool
	closedErr     error
	token         string
}

// ResetClient updates the client's state with new connection and context information.
func (c *Client) ResetClient(ctx *gin.Context, conn *websocket.Conn, manager *ClientManager) {
	c.w = new(sync.Mutex)
	c.conn = conn
	//c.PlatformID = stringutil.StringToInt(ctx.GetPlatformID())
	//c.IsCompress = ctx.GetCompression()
	//c.IsBackground = ctx.GetBackground()
	//c.UserID = ctx.Get("userID").(string)
	c.UserID = ctx.GetString("userID")
	c.ctx = ctx
	c.clientManager = manager
	c.closed.Store(false)
	c.closedErr = nil
	//c.token = ctx.GetToken()
}

func pongWaitTime() time.Time {
	return time.Now().Add(pongWait)
}

func (c *Client) pingHandler(_ string) error {
	if err := c.conn.SetReadDeadline(pongWaitTime()); err != nil {
		return err
	}

	return c.writePongMsg()
}

// readMessage continuously reads messages from the connection.
func (c *Client) readMessage() {
	defer func() {
		if r := recover(); r != nil {
			c.closedErr = ErrPanic
			glog.Error("socket have panic err:", r, string(debug.Stack()))
		}
		c.close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(pongWaitTime())
	c.conn.SetPingHandler(c.pingHandler)

	for {
		glog.Info("readMessage")
		messageType, message, returnErr := c.conn.ReadMessage()
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
			_ = c.conn.SetReadDeadline(pongWaitTime())
			parseDataErr := c.handleMessage(message)
			if parseDataErr != nil {
				c.closedErr = parseDataErr
				return
			}
		case MessageText:
			//c.closedErr = ErrNotSupportMessageProtocol
			//return
			_ = c.conn.SetReadDeadline(pongWaitTime())
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
		resp, messageErr = c.handleHeartBeat(c.ctx, binaryReq)
	} else {
		resp, messageErr = BinaryMessageHandler(c, binaryReq)
	}

	return c.replyMessage(binaryReq, messageErr, resp)
}

func (c *Client) handleHeartBeat(ctx context.Context, req *Req) ([]byte, error) {
	return []byte(req.Data), nil
}

func (c *Client) close() {
	if c.closed.Load() {
		return
	}

	c.w.Lock()
	defer c.w.Unlock()

	c.closed.Store(true)
	c.conn.Close()
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

func (c *Client) writeBinaryMsg(resp Resp) error {
	if c.closed.Load() {
		return nil
	}

	encodedBuf := resp.String()

	c.w.Lock()
	defer c.w.Unlock()

	err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(MessageBinary, []byte(encodedBuf))
}

func (c *Client) writePongMsg() error {
	if c.closed.Load() {
		return nil
	}

	c.w.Lock()
	defer c.w.Unlock()

	err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(PongMessage, nil)
}
