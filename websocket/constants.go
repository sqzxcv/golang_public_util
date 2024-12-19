package websocket

import "time"

const (
	// Websocket Protocol.
	WSGetNewestSeq        = 1001
	WSPullMsgBySeqList    = 1002
	WSSendMsg             = 1003
	WSSendSignalMsg       = 1004
	WSPushMsg             = 2001
	WSKickOnlineMsg       = 2002
	WsLogoutMsg           = 2003
	WsSetBackgroundStatus = 2004
	WSDataError           = 3001
	WsHeartbeat           = 5001
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 30 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 51200
)
