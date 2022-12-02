package MysqlErr

import mysql2 "github.com/go-sql-driver/mysql"

var (
    // ErrDuplicateEntryCode 命中唯一索引
    ErrDuplicateEntryCode = 1062
)

// MysqlErrCode 根据mysql错误信息返回错误代码
func MysqlErrCode(err error) int {
    mysqlErr, ok := err.(*mysql2.MySQLError)
    if !ok {
        return 0
    }
    return int(mysqlErr.Number)
}
