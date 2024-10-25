package base

import "github.com/Logiase/MiraiGo-Template/global/config"

var (
	PasswordHash [16]byte // 存储QQ密码哈希供登录使用
	AccountToken []byte   // 存储 AccountToken 供登录使用

	SignServers       []config.SignServer // 签名服务器
	SignServerTimeout uint                // 签名服务器超时时间

	Account *config.Account // 账户配置
)

func Init() {
	conf := config.Parse("./config.yml")
	{
		Account = conf.Account
		SignServers = conf.Account.SignServers
		SignServerTimeout = conf.Account.SignServerTimeout
	}
}
