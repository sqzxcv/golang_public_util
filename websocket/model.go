package websocket

import (
	"encoding/json"
	"sync"
)

type Req struct {
	ReqIdentifier int32  `json:"reqIdentifier" validate:"required"`
	Token         string `json:"token"`
	SendID        string `json:"sendID"        validate:"required"`
	OperationID   string `json:"operationID"   validate:"required"`
	MsgIncr       string `json:"msgIncr"       validate:"required"`
	Data          string `json:"data"`
}

func (r *Req) String() string {
	var tReq Req
	tReq.ReqIdentifier = r.ReqIdentifier
	tReq.Token = r.Token
	tReq.SendID = r.SendID
	tReq.OperationID = r.OperationID
	tReq.MsgIncr = r.MsgIncr
	tReq.Data = r.Data
	return structToJsonString(tReq)
}

func structToJsonString(param any) string {
	m, _ := json.Marshal(param)
	dataString := string(m)
	return dataString
}

var reqPool = sync.Pool{
	New: func() any {
		return new(Req)
	},
}

func getReq() *Req {
	req := reqPool.Get().(*Req)
	req.Data = ""
	req.MsgIncr = ""
	req.OperationID = ""
	req.ReqIdentifier = 0
	req.SendID = ""
	req.Token = ""
	return req
}

func (r *Req) DecodeFromMsg(msg []byte) error {
	err := json.Unmarshal(msg, r)
	if err != nil {
		return err
	}
	return nil
}

func freeReq(req *Req) {
	reqPool.Put(req)
}

type Resp struct {
	ReqIdentifier int32  `json:"reqIdentifier"`
	MsgIncr       string `json:"msgIncr"`
	OperationID   string `json:"operationID"`
	ErrCode       int    `json:"errCode"`
	ErrMsg        string `json:"errMsg"`
	Data          string `json:"data"`
}

func (r *Resp) String() string {
	var tResp Resp
	tResp.ReqIdentifier = r.ReqIdentifier
	tResp.MsgIncr = r.MsgIncr
	tResp.OperationID = r.OperationID
	tResp.ErrCode = r.ErrCode
	tResp.ErrMsg = r.ErrMsg
	tResp.Data = r.Data
	return structToJsonString(tResp)
}
