package exception

import (
	"runtime/debug"

	"github.com/sqzxcv/glog"
)

// CatchException 捕获异常
func CatchException(message string) {

	if err := recover(); err != nil {
		glog.Error(message, "[异常=============] ", err, "\n", string(debug.Stack()))
	}
	glog.Flush()
}
