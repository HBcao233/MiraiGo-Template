package base

import (
	"time"

	"github.com/Logiase/MiraiGo-Template/global/config"
	"gopkg.in/yaml.v3"
)

var (
	SplitURL         = false // 是否分割URL
	AllowTempSession = false // 允许临时会话
	SkipMimeScan     = false // 是否跳过Mime扫描
	ConvertWebpImage = false // 是否转换Webp图片

	PasswordHash [16]byte // 存储QQ密码哈希供登录使用
	AccountToken []byte   // 存储 AccountToken 供登录使用

	SignServers       []config.SignServer // 签名服务器
	SignServerTimeout uint                // 签名服务器超时时间

	Account *config.Account // 账户配置

	PostFormat        = "string"        // 上报格式 string or array
	HeartbeatInterval = time.Second * 5 // 心跳间隔

	Database map[string]yaml.Node // 数据库列表
)

func Init() {
	conf := config.Parse("./config.yml")
	{
		Account = conf.Account
		SignServers = conf.Account.SignServers
		SignServerTimeout = conf.Account.SignServerTimeout
	}
}
