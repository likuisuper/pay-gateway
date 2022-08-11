package code

//配置数据库连接名
const (
	MYSQL_READING = "cloud_read"
)

// 错误码
const (
	CODE_OK    = 2000 //成功
	CODE_ERROR = 1005 //操作失败(用户toast)    无上报

)

//用于gorm Sum返回
type GormSumTotal struct {
	SumTotal int `json:"sum_total"`
}
