package AutoMigrationDB

import (
	"gorm.io/gorm/schema"
)

// MigrationTable 数据库表升级记录映射
// key: 实现了schema.Tabler接口的表结构体
// value: 表版本号，用于控制表结构升级
var MigrationTable = map[schema.Tabler]int{}

// InitTask 初始化任务映射
// key: 任务名称（通常对应系统功能模块）
// value: 任务版本号，用于控制任务是否需要执行
var InitTask = map[string]int{}

// InitTaskAction 初始化任务执行函数映射
// key: 任务名称（与InitTask中的key对应）
// value: 具体的初始化执行函数，返回错误信息
var InitTaskAction = map[string]func() error{}
