package AutoMigrationDB

import (
	"github.com/sqzxcv/glog"
	"gorm.io/gorm"

	"time"
)

type Migration struct {
	Table    string    `gorm:"column:table;primaryKey;not null" json:"table"`
	Version  int       `gorm:"column:version;not null" json:"version"`
	UpdateAt time.Time `gorm:"column:updateAt;not null" json:"updateAt"`
}

// TableName V2ServerGroup's table name
func (Migration) TableName() string {
	return "migration"
}

func AutoMigrationDB(db *gorm.DB) error {

	if db.Migrator().HasTable(&Migration{}) == false {
		err := db.AutoMigrate(&Migration{})
		if err != nil {
			glog.Error("创建 Migration 表失败, ", err.Error())
			return err

		}
		glog.Info("Migration create successfully")
	}
	tx := db.Begin()
	//tx = tx.Exec("SET FOREIGN_KEY_CHECKS=0;")
	//defer tx.Exec("SET FOREIGN_KEY_CHECKS=1;")
	var allTables []Migration
	err := tx.Find(&allTables).Error
	if err != nil {
		glog.Error("查询 Migration 表失败, ", err.Error())
		return err
	}
	tableMap := make(map[string]Migration)
	for _, table := range allTables {
		tableMap[table.Table] = table
	}

	for table, version := range MigrationTable {
		m, ok := tableMap[table.TableName()]
		needSave := false
		if !ok {
			err = tx.AutoMigrate(table)
			if err != nil {
				glog.FError("创建表[%s]失败, 原因:%s", table.TableName(), err.Error())
				tx.Rollback()
				return err
			}
			needSave = true
		} else if m.Version < version {

			err = tx.AutoMigrate(table)
			if err != nil {
				glog.FError("升级表[%s]失败, 原因:%s", table.TableName(), err.Error())
				tx.Rollback()
				return err
			}
			needSave = true
		}
		if needSave {
			m.Table = table.TableName()
			m.Version = version
			m.UpdateAt = time.Now()
			err = tx.Save(&m).Error
			if err != nil {
				glog.FError("更新 Migration 表失败, 原因:%s", err.Error())
				tx.Rollback()
				return err
			}
			glog.FInfo("表[%s]更新成功", table.TableName())
		}

	}

	err = tx.Commit().Error
	if err != nil {
		glog.Error("Failed to commit tx", err)
		tx.Rollback()
		return err
	}

	tx = db.Begin()
	for action, version := range InitTask {
		m, ok := tableMap[action]
		needProcess := false
		if !ok {
			needProcess = true
		} else if m.Version < version {

			needProcess = true
		}
		if needProcess {
			// 开始执行task, 执行成功保存, 否则不保存
			err = InitTaskAction[action]()
			if err != nil {
				glog.FError("Action[%s]执行失败, 原因:%s", action, err.Error())
				tx.Rollback()
				return err
			}
			m.Table = action
			m.Version = version
			m.UpdateAt = time.Now()
			err = tx.Save(&m).Error
			if err != nil {
				glog.FError("更新 Migration 表失败, 原因:%s", err.Error())
				tx.Rollback()
				return err
			}
			glog.FInfo("Action[%s]执行成功", action)
		}
	}
	err = tx.Commit().Error
	if err != nil {
		glog.Error("Failed to commit tx", err)
		tx.Rollback()
		return err
	}
	return nil
}
