package bot

import (
	"bufio"
	"bytes"
	"fmt"
	_ "image/png"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/pkg/errors"

	qrcodeTerminal "github.com/Baozisoftware/qrcode-terminal-go"
	"github.com/tuotoo/qrcode"

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
}

func LoginWithOption(option LoginOption) error {
	if option.Token != nil {
		err := func() error {
			logger.Infof("检测到会话缓存, 尝试快速恢复登录")
			var token = option.Token
			r := binary.NewReader(token)
			cu := r.ReadInt64()
			if Instance.Uin != 0 {
				if cu != Instance.Uin && !option.UseTokenWhenUnmatchedUin {
					return fmt.Errorf("配置文件内的QQ号 (%v) 与会话缓存内的QQ号 (%v) 不相同", Instance.Uin, cu)
				}
			}
			if err := Instance.TokenLogin(token); err != nil {
				time.Sleep(time.Second)
				Instance.Disconnect()
				return errors.Errorf("恢复会话失败(%s)", err)
			} else {
				logger.Infof("快速恢复登录成功")
				return nil
			}
		}()
		if err != nil {
			logger.WithError(err).Warn("failed restore session by token, fallback to common or qrcode")
		} else {
			return nil
		}
	}
	switch option.LoginMethod {
	case LoginMethodCommon:
		return CommonLogin()
	case LoginMethodQRCode:
		return QrcodeLogin()
	default:
		return errors.New("unknown login method")
	}
}

// CommonLogin 普通账号密码登录
func CommonLogin() error {
	res, err := Instance.Login()
	if err != nil {
		return err
	}
	return loginResponseProcessor(res)
}

// QrcodeLogin 扫码登陆
func QrcodeLogin() error {
	rsp, err := Instance.FetchQRCodeCustomSize(3, 4, 2)
	if err != nil {
		return err
	}
	fi, err := qrcode.Decode(bytes.NewReader(rsp.ImageData))
	if err != nil {
		return err
	}
	_ = os.WriteFile("qrcode.png", rsp.ImageData, 0o644)
	defer func() { _ = os.Remove("qrcode.png") }()
	if Instance.Uin != 0 {
		logger.Infof("请使用账号 %v 登录手机QQ扫描二维码 (qrcode.png) : ", Instance.Uin)
	} else {
		logger.Infof("请使用手机QQ扫描二维码 (qrcode.png) : ")
	}
	time.Sleep(time.Second)
	qrcodeTerminal.New().Get(fi.Content).Print()
	s, err := Instance.QueryQRCodeStatus(rsp.Sig)
	if err != nil {
		return err
	}
	prevState := s.State
	for {
		time.Sleep(time.Second)
		s, _ = Instance.QueryQRCodeStatus(rsp.Sig)
		if s == nil {
			continue
		}
		if prevState == s.State {
			continue
		}
		prevState = s.State
		switch s.State {
		case client.QRCodeCanceled:
			logger.Fatalf("扫码被用户取消.")
		case client.QRCodeTimeout:
			logger.Fatalf("二维码过期")
		case client.QRCodeWaitingForConfirm:
			logger.Infof("扫码成功, 请在手机端确认登录.")
		case client.QRCodeConfirmed:
			res, err := Instance.QRCodeLogin(s.LoginInfo)
			if err != nil {
				return err
			}
			return loginResponseProcessor(res)
		case client.QRCodeImageFetch, client.QRCodeWaitingForScan:
			// ignore
		}
	}
}

// ErrSMSRequestError SMS请求出错
var ErrSMSRequestError = errors.New("sms request error")

var console = bufio.NewReader(os.Stdin)

func readLine() (str string) {
	str, _ = console.ReadString('\n')
	str = strings.TrimSpace(str)
	return
}

func readLineTimeout(t time.Duration, de string) (str string) {
	r := make(chan string)
	go func() {
		select {
		case r <- readLine():
		case <-time.After(t):
		}
	}()
	str = de
	select {
	case str = <-r:
	case <-time.After(t):
	}
	return
}

// loginResponseProcessor 登录结果处理
func loginResponseProcessor(res *client.LoginResponse) error {
	var err error
	for {
		if err != nil {
			return err
		}
		if res.Success {
			return nil
		}
		var text string
		switch res.Error {
		case client.SliderNeededError:
			logger.Warnf("登录需要滑条验证码, 请验证:")
			ticket := getTicket(res.VerifyUrl)
			if ticket == "" {
				logger.Infof("按 Enter 继续....")
				readLine()
				os.Exit(0)
			}
			res, err = Instance.SubmitTicket(ticket)
			continue
		case client.NeedCaptcha:
			logger.Warnf("登录需要验证码.")
			_ = os.WriteFile("captcha.jpg", res.CaptchaImage, 0o644)
			logger.Warnf("请输入验证码 (captcha.jpg)： (Enter 提交)")
			text = readLine()
			_ = os.Remove("captcha.jpg")
			res, err = Instance.SubmitCaptcha(text, res.CaptchaSign)
			continue
		case client.SMSNeededError:
			logger.Warnf("账号已开启设备锁, 按 Enter 向手机 %v 发送短信验证码.", res.SMSPhone)
			readLine()
			if !Instance.RequestSMS() {
				logger.Warnf("发送验证码失败，可能是请求过于频繁.")
				return errors.WithStack(ErrSMSRequestError)
			}
			logger.Warn("请输入短信验证码： (Enter 提交)")
			text = readLine()
			res, err = Instance.SubmitSMS(text)
			continue
		case client.SMSOrVerifyNeededError:
			logger.Warnf("账号已开启设备锁，请选择验证方式:")
			logger.Warnf("1. 向手机 %v 发送短信验证码", res.SMSPhone)
			logger.Warnf("2. 使用手机QQ扫码验证.")
			logger.Warn("请输入(1 - 2) (将在10秒后自动选择2)：")
			text = readLineTimeout(time.Second*10, "2")
			if strings.Contains(text, "1") {
				if !Instance.RequestSMS() {
					logger.Warnf("发送验证码失败，可能是请求过于频繁.")
					return errors.WithStack(ErrSMSRequestError)
				}
				logger.Warn("请输入短信验证码： (Enter 提交)")
				text = readLine()
				res, err = Instance.SubmitSMS(text)
				continue
			}
			fallthrough
		case client.UnsafeDeviceError:
			logger.Warnf("账号已开启设备锁，请前往 -> %v <- 验证后重启Bot.", res.VerifyUrl)
			logger.Infof("按 Enter 或等待 5s 后继续....")
			readLineTimeout(time.Second*5, "")
			os.Exit(0)
		case client.OtherLoginError, client.UnknownLoginError, client.TooManySMSRequestError:
			msg := res.ErrorMessage
			if strings.Contains(msg, "版本") {
				msg = "密码错误或账号被冻结"
			}
			if strings.Contains(msg, "冻结") {
				logger.Fatalf("账号被冻结")
			}
			logger.Warnf("登录失败: %v", msg)
			logger.Infof("按 Enter 或等待 5s 后继续....")
			readLineTimeout(time.Second*5, "")
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
