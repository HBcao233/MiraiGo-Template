package bot

import (
	"fmt"
	_ "image/png"
	"os"
	"path"

	"github.com/Logiase/MiraiGo-Template/global"
	"github.com/Logiase/MiraiGo-Template/global/base"
	"github.com/Logiase/MiraiGo-Template/global/cache"
	"github.com/Logiase/MiraiGo-Template/global/coolq"
	"github.com/Logiase/MiraiGo-Template/global/db"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/wrapper"
	"github.com/mattn/go-colorable"

	log "github.com/sirupsen/logrus"
)

var cli *client.QQClient
var device *client.DeviceInfo
var handlers []func(*coolq.CQBot, *coolq.Event)
var hooks_started []func()
var hooks_connected []func(*coolq.CQBot)

func newClient() *client.QQClient {
	c := client.NewClientEmpty()
	// c.UseFragmentMessage = base.ForceFragmented
	c.OnServerUpdated(func(_ *client.QQClient, _ *client.ServerUpdatedEvent) bool {
		// if !base.UseSSOAddress {
		// 	log.Infof("收到服务器地址更新通知, 根据配置文件已忽略.")
		// 	return false
		// }
		log.Infof("收到服务器地址更新通知, 将在下一次重连时应用. ")
		return true
	})
	if global.PathExists("address.txt") {
		log.Infof("检测到 address.txt 文件. 将覆盖目标IP.")
		addr := global.ReadAddrFile("address.txt")
		if len(addr) > 0 {
			c.SetCustomServer(addr)
		}
		log.Infof("读取到 %v 个自定义地址.", len(addr))
	}
	// c.SetLogger(protocolLogger{})
	return c
}

// Init 初始化
func Init() {
	base.Init()
	log.SetOutput(colorable.NewColorableStdout())
	log.SetFormatter(&global.LogFormat{EnableColor: true})
	loadDevice()
	loadSignServer()
	loadVersions()

	mkCacheDir := func(path string, _type string) {
		if !global.PathExists(path) {
			if err := os.MkdirAll(path, 0o755); err != nil {
				log.Fatalf("创建%s缓存文件夹失败: %v", _type, err)
			}
		}
	}
	mkCacheDir(global.ImagePath, "图片")
	mkCacheDir(global.VoicePath, "语音")
	mkCacheDir(global.VideoPath, "视频")
	mkCacheDir(global.CachePath, "发送图片")
	mkCacheDir(path.Join(global.ImagePath, "guild-images"), "频道图片缓存")
	mkCacheDir(global.VersionsPath, "版本缓存")
	cache.Init()

	db.Init()
	if err := db.Open(); err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	for _, f := range hooks_started {
		go f()
	}
}

func loadDevice() {
	if !global.PathExists("device.json") {
		log.Warn("虚拟设备信息不存在, 将自动生成随机设备.")
		device = client.GenRandomDevice()
		_ = os.WriteFile("device.json", device.ToJson(), 0o644)
		log.Info("已生成设备信息并保存到 device.json 文件.")
	} else {
		log.Info("将使用 device.json 内的设备信息运行Bot.")
		device = new(client.DeviceInfo)
		if err := device.ReadJson([]byte(global.ReadAllText("device.json"))); err != nil {
			log.Fatalf("加载设备信息失败: %v", err)
		}
	}
}

func loadSignServer() {
	signServer, err := getAvaliableSignServer() // 获取可用签名服务器
	if err != nil {
		log.Warn(err)
	}
	if signServer != nil && len(signServer.URL) > 1 {
		log.Infof("使用签名服务器：%v", signServer.URL)
		// go signStartRefreshToken(conf.Account.RefreshInterval) // 定时刷新 token
		wrapper.DandelionEnergy = energy
		wrapper.FekitGetSign = sign
		// if !conf.IsBelow110 {
		// 	if !conf.Account.AutoRegister {
		// 		log.Warn("自动注册实例已关闭，请配置 sign-server 端自动注册实例以保持正常签名")
		// 	}
		// 	if !conf.Account.AutoRefreshToken {
		// 		log.Info("自动刷新 token 已关闭，token 过期后获取签名时将不会立即尝试刷新获取新 token")
		// 	}
		// } else {
		// 	log.Warn("签名服务器版本 <= 1.1.0 ，无法使用刷新 token 等操作，建议使用 1.1.6 版本及以上签名服务器")
		// }
	} else {
		log.Warnf("警告: 未配置签名服务器或签名服务器不可用, 这可能会导致登录 45 错误码或发送消息被风控")
	}
}

func loadVersions() {
	// 加载本地版本信息, 一般是在上次登录时保存的
	versionFile := path.Join(global.VersionsPath, fmt.Sprint(int(device.Protocol))+".json")
	if global.PathExists(versionFile) {
		b, err := os.ReadFile(versionFile)
		if err != nil {
			log.Warnf("从文件 %s 读取本地版本信息文件出错.", versionFile)
			os.Exit(0)
		}
		err = device.Protocol.Version().UpdateFromJson(b)
		if err != nil {
			log.Warnf("从文件 %s 解析本地版本信息出错: %v", versionFile, err)
			os.Exit(0)
		}
		log.Infof("从文件 %s 读取协议版本 %v.", versionFile, device.Protocol.Version())
	}
}

func GetBot() *client.QQClient {
	return cli
}
func AddHandler(f func(*coolq.CQBot, *coolq.Event)) {
	handlers = append(handlers, f)
}

func OnStart(f func()) {
	hooks_started = append(hooks_started, f)
}

func OnConnect(f func(*coolq.CQBot)) {
	hooks_connected = append(hooks_connected, f)
}
