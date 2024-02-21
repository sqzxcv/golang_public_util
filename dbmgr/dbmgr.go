package dbmgr

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "github.com/go-redis/redis/v8"
    "go.mongodb.org/mongo-driver/mongo/options"
    opt "github.com/qiniu/qmgo/options"
    "gorm.io/driver/mysql"
    "strings"

    "github.com/sqzxcv/glog"

    "github.com/qiniu/qmgo"
    "go.mongodb.org/mongo-driver/mongo"
    "gorm.io/gorm"
    "go.mongodb.org/mongo-driver/event"
)

// MySqlConfig 系统配置
type MySqlConfig struct {
    DBHost     string `mapstructure:"db_host"`
    DBPort     string `mapstructure:"db_port"`
    DBAccount  string `mapstructure:"db_account"`
    DBPassword string `mapstructure:"db_password"`
    DBName     string `mapstructure:"db_name"`
    DBLinkInfo string `mapstructure:"db_link_info"`
}

type RedisConfig struct {
    DBAddr     string `mapstructure:"db_addr"`
    DBPassword string `mapstructure:"db_password"`
    DBNum      int    `mapstructure:"db_num"`
}

type MongoDBConfig struct {
    Path string `mapstructure:"path" json:"path" yaml:"path"` // 服务器地址
    Port string `mapstructure:"port" json:"port" yaml:"port"` //:端口
}

type QmgoConfig struct {
    Path   string `mapstructure:"path" json:"path" yaml:"path"`          // 服务器地址
    Port   string `mapstructure:"port" json:"port" yaml:"port"`          //:端口
    DBName string `mapstructure:"db_name" json:"db_name" yaml:"db_name"` //:数据库名称
    DBAccount  string `mapstructure:"db_account"`
    DBPassword string `mapstructure:"db_password"`
    AuthenticationDatabase string `mapstructure:"authentication_database"`
    ShowDebug bool `mapstructure:"show_debug"`
}

type Mgr struct {
    // config
    MySqlConf *MySqlConfig

    RedisConf   *RedisConfig
    MongoDBConf *MongoDBConfig
    QmgoConf    *QmgoConfig

    // mysql
    DB *gorm.DB

    RedisDB      *redis.Client
    MongoDBEngin *mongo.Client

    QmgoDB *qmgo.Database
}

var Dbmgr *Mgr

func init() {
}

//const RedisPrefix = "GaGa_redis_prefix_"

func NewDBMgr(mysqlConfig *MySqlConfig, redisConfig *RedisConfig, mongoConf *MongoDBConfig, qmgoConf *QmgoConfig) (dbmgr *Mgr, err error) {
    dbmgr = &Mgr{}
    dbmgr.MySqlConf = mysqlConfig
    dbmgr.RedisConf = redisConfig
    dbmgr.MongoDBConf = mongoConf
    dbmgr.QmgoConf = qmgoConf

    if dbmgr.MySqlConf != nil {
        str := dbmgr.MySqlConf.DBLinkInfo
        if len(str) == 0 {
            str = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?timeout=90s&parseTime=true", dbmgr.MySqlConf.DBAccount, dbmgr.MySqlConf.DBPassword, dbmgr.MySqlConf.DBHost, dbmgr.MySqlConf.DBPort, dbmgr.MySqlConf.DBName)
        }

        db, err := gorm.Open(mysql.Open(str), &gorm.Config{})

        if err != nil {
            glog.Error("打开数据库失败, 原因" + err.Error())
        }

        dbmgr.DB = db
    }

    if dbmgr.RedisConf != nil {
        Rlient := redis.NewClient(&redis.Options{
            Addr:     dbmgr.RedisConf.DBAddr,
            Password: dbmgr.RedisConf.DBPassword, // no password set
            DB:       dbmgr.RedisConf.DBNum,      // use default DB
        })
        ctx := context.Background()
        pong, err := Rlient.Ping(ctx).Result()
        if err != nil {
            glog.Error("RedisDB 连接失败, 原因:", err.Error())
            panic(err)
        }
        glog.Info("RedisDB 连接成功", pong)
        dbmgr.RedisDB = Rlient
    }

    if dbmgr.MongoDBConf != nil {
        var err error
        clientOptions := options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s", dbmgr.MongoDBConf.Path, dbmgr.MongoDBConf.Port))

        // 连接到MongoDB
        mgoCli, err := mongo.Connect(context.TODO(), clientOptions)
        if err != nil {
            glog.Error("创建MongoDB 失败, 原因:", err)
            panic(err)
        }
        // 检查连接
        err = mgoCli.Ping(context.TODO(), nil)
        if err != nil {
            glog.Error("检测 MongoDB 链接失败, 原因:", err)
            panic(err)
        }
        dbmgr.MongoDBEngin = mgoCli
    }

    if dbmgr.QmgoConf != nil {
        monitor := &event.CommandMonitor{
            Started: func(_ context.Context, e *event.CommandStartedEvent) {
                glog.Debug(e.Command)
            },
            Succeeded: func(_ context.Context, e *event.CommandSucceededEvent) {
                glog.Debug(e.Reply)
            },
            Failed: func(_ context.Context, e *event.CommandFailedEvent) {
                glog.Error(e.Failure)
            },
        }

        clientOptions := opt.ClientOptions{//<--注意：这个options.ClientOptions是qmgo自己封装的类型，里面继承了官方的
            ClientOptions: &options.ClientOptions{//这个opt是mongoDrive官方的options，我给他起别名为opt
                Monitor: monitor,
            },
        }
        ctx := context.Background()
        uri := fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", dbmgr.QmgoConf.DBAccount, dbmgr.QmgoConf.DBPassword, dbmgr.QmgoConf.Path, dbmgr.QmgoConf.Port, dbmgr.QmgoConf.DBName)
        if len(dbmgr.QmgoConf.AuthenticationDatabase) != 0 {
            uri = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s&authMechanism=SCRAM-SHA-1", dbmgr.QmgoConf.DBAccount, dbmgr.QmgoConf.DBPassword, dbmgr.QmgoConf.Path, dbmgr.QmgoConf.Port, dbmgr.QmgoConf.DBName, dbmgr.QmgoConf.AuthenticationDatabase)
        }
        cfg := &qmgo.Config{Uri: uri}
        var client *qmgo.Client
        if dbmgr.QmgoConf.ShowDebug {
            client, err = qmgo.NewClient(ctx, cfg, clientOptions)
        } else {
            client, err = qmgo.NewClient(ctx, cfg)
        }
        if err != nil {
            glog.Error("创建MongoDB 失败, 原因:", err)
            panic(err)
        }
        // 检查连接
        err = client.Ping(60)
        if err != nil {
            glog.Error("检测 MongoDB 链接失败, 原因:", err)
            panic(err)
        }
        if len(dbmgr.QmgoConf.DBName) == 0 {
            err := errors.New("请指定MongoDB的Database name")
            glog.Error("创建MongoDB 失败, 原因:", err)
            panic(err)
        }
        db := client.Database(dbmgr.QmgoConf.DBName)
        dbmgr.QmgoDB = db
    }

    return dbmgr, err
}

// Close session
func (mgr *Mgr) Close() {

    return
}

// IsQueryNoItemError 通过检查err最后是不是以"no rows in result set"结尾, 来判断这个错误是不是没有搜索到结果引起的;只适用于QueryRow(sql).Scan
func (mgr *Mgr) IsQueryNoItemError(err error) (result bool) {

    if err == sql.ErrNoRows || strings.HasSuffix(err.Error(), "no rows in result set") {
        result = true
    } else {
        result = false
    }
    return result
}
