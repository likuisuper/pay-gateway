package db

import (
	"fmt"
	"os"

	"github.com/zeromicro/go-zero/core/logx"
	redisdb "gitlab.muchcloud.com/consumer-project/zhuyun-core/cache"
	mysql "gitlab.muchcloud.com/consumer-project/zhuyun-core/db"
	"gorm.io/gorm"

	"strings"
)

func DBInit(mysqlCfgs []*mysql.DbConfig, redisCfgs []*redisdb.RedisConfigs) {
	err := mysql.InitMysql(mysqlCfgs)
	if err != nil {
		logx.Errorf("mysql init YunYueDu err, err= %v", err)
		os.Exit(1)
	}

	err = redisdb.InitRedis(redisCfgs)
	if err != nil {
		os.Exit(1)
	}

	rdbCount := 0
	for _, rdbCfg := range redisCfgs {
		rdbCount += len(rdbCfg.DBs)
	}
	logx.Infof("init redis success! redis cfg count:%d instance count:%d", len(redisCfgs), rdbCount)
}

/*
 * 不需要trace ctx 传nil
 * mysql gorm 携带ctx
 *name 需要的数据库连接
 */
func WithDBContext(name string) *gorm.DB {
	instance, err := mysql.GetMysqlInstance(name)
	if err != nil {
		logx.Errorf("get mysql [ %s ] instance err, err= %v", name, err)
	}

	return instance.DB
}

func WithRedisDBContext(name string) *redisdb.RedisInstance {
	instance, err := redisdb.GetRedisInstance(name)
	if err != nil {
		logx.Errorf("get redis instance error, err= %v", err)
	}
	return instance
}

// mysql字段自增或自减方法
func IncrementOrDecrementField(Db *gorm.DB, tableName string, where map[string]interface{}, files map[string]int) (int64, error) {
	//组装mysql语句
	sqlString := make([]string, 0)
	sqlString = append(sqlString, "UPDATE `")
	sqlString = append(sqlString, tableName)
	sqlString = append(sqlString, "` SET ")
	//修改值拼装
	strFiles := make([]string, 0)
	for k, v := range files {
		strFiles = append(strFiles, fmt.Sprintf("`%s` = `%s`+ (%d)", k, k, v))
	}
	sqlString = append(sqlString, strings.Join(strFiles, ","))
	//条件拼装
	i := 0
	str := make([]string, len(where))
	val := make([]interface{}, len(where))
	for k, v := range where {
		whereArr := strings.Split(k, ":")
		if len(whereArr) > 1 {
			str[i] = fmt.Sprintf("`%s` %s  ? ", whereArr[0], whereArr[1])
		} else {
			str[i] = "`" + k + "` = ? "
		}
		val[i] = v
		i++
	}
	sqlString = append(sqlString, " WHERE ")
	sqlString = append(sqlString, strings.Join(str, " AND "))
	sql := strings.Join(sqlString, "")
	gormDb := Db.Exec(sql, val...)
	if gormDb.Error != nil {
		logx.Errorf("update %s fail, sql= %s, val= %v, error= %v", tableName, sql, val, gormDb.Error)
	}
	return gormDb.RowsAffected, gormDb.Error
}
