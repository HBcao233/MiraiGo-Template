package bot

import (
	"bufio"
	"crypto/md5"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Logiase/MiraiGo-Template/global"
	"github.com/Logiase/MiraiGo-Template/internal/base"
	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.ilharper.com/x/isatty"
)

var console = bufio.NewReader(os.Stdin)
var ErrSMSRequestError = errors.New("sms request error")

func readLine() (str string) {
	str, _ = console.ReadString('\n')
	str = strings.TrimSpace(str)
	return
}

func readLineTimeout(t time.Duration) {
	r := make(chan string)
	go func() {
		select {
		case r <- readLine():
		case <-time.After(t):
		}
	}()
	select {
	case <-r:
	case <-time.After(t):
	}
}

func readIfTTY(de string) (str string) {
	if isatty.Isatty(os.Stdin.Fd()) {
		return readLine()
	}
	log.Warnf("未检测到输入终端，自动选择%s.", de)
	return de
}

// Login 登录
func Login() {
	if len(base.Account.Password) > 0 {
		base.PasswordHash = md5.Sum([]byte(base.Account.Password))
	}

	log.Info("开始尝试登录并同步消息...")
	log.Infof("使用协议: %s", device.Protocol.Version())
	cli = newClient()
	cli.UseDevice(device)
	isQRCodeLogin := (base.Account.Uin == 0 || len(base.Account.Password) == 0) && !base.Account.Encrypt
	isTokenLogin := false

	if isQRCodeLogin && cli.Device().Protocol != 2 {
		log.Warn("当前协议不支持二维码登录, 请配置账号密码登录.")
		os.Exit(0)
	}

	if global.PathExists("session.token") {
		token := tokenLogin()
		if token != nil {
			isTokenLogin = true
		}
	}
	if base.Account.Uin != 0 && base.PasswordHash != [16]byte{} {
		cli.Uin = base.Account.Uin
		cli.PasswordMd5 = base.PasswordHash
	}
	if !isTokenLogin {
		if !isQRCodeLogin {
			if err := commonLogin(); err != nil {
				log.Fatalf("登录时发生致命错误: %v", err)
			}
		}
	}
	var reLoginLock sync.Mutex
	cli.DisconnectedEvent.Subscribe(func(_ *client.QQClient, e *client.ClientDisconnectedEvent) {
		reLoginLock.Lock()
		defer reLoginLock.Unlock()
		if cli.Online.Load() {
			return
		}
		log.Warnf("Bot已离线: %v", e.Message)
		log.Warnf("未启用自动重连, 将退出.")
		os.Exit(1)
	})

	saveToken()
	cli.AllowSlider = true
	log.Infof("登录成功 欢迎使用: %v", cli.Nickname)
	log.Info("开始加载好友列表...")
	global.Check(cli.ReloadFriendList(), true)
	log.Infof("共加载 %v 个好友.", len(cli.FriendList))
	log.Infof("开始加载群列表...")
	global.Check(cli.ReloadGroupList(), true)
	log.Infof("共加载 %v 个群.", len(cli.GroupList))
	log.Info("资源初始化完成, 开始处理信息.")
	log.Info("アトリは、高性能ですから!")
}

func saveToken() {
	base.AccountToken = cli.GenToken()
	_ = os.WriteFile("session.token", base.AccountToken, 0o644)
}

func tokenLogin() []byte {
	token, err := os.ReadFile("session.token")
	if err == nil {
		if base.Account.Uin != 0 {
			r := binary.NewReader(token)
			cu := r.ReadInt64()
			if cu != base.Account.Uin {
				log.Warnf("警告: 配置文件内的QQ号 (%v) 与缓存内的QQ号 (%v) 不相同", base.Account.Uin, cu)
				log.Warnf("1. 使用会话缓存继续.")
				log.Warnf("2. 删除会话缓存并重启.")
				log.Warnf("请选择:")
				text := readIfTTY("1")
				if text == "2" {
					_ = os.Remove("session.token")
					log.Infof("缓存已删除.")
					os.Exit(0)
				}
			}
		}
		if err = cli.TokenLogin(token); err != nil {
			_ = os.Remove("session.token")
			log.Warnf("恢复会话失败: %v , 尝试使用正常流程登录.", err)
			time.Sleep(time.Second)
			cli.Disconnect()
			cli.Release()
			cli = newClient()
			cli.UseDevice(device)
			return nil
		}
		return token
	}
	return nil
}

func commonLogin() error {
	res, err := cli.Login()
	if err != nil {
		return err
	}
	return loginResponseProcessor(res)
}

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
			log.Warnf("登录需要滑条验证码, 请验证后重试.")
			ticket := getTicket(res.VerifyUrl)
			if ticket == "" {
				log.Infof("按 Enter 继续....")
				readLine()
				os.Exit(0)
			}
			res, err = cli.SubmitTicket(ticket)
			continue
		case client.NeedCaptcha:
			log.Warnf("登录需要验证码.")
			_ = os.WriteFile("captcha.jpg", res.CaptchaImage, 0o644)
			log.Warnf("请输入验证码 (captcha.jpg)： (Enter 提交)")
			text = readLine()
			global.DelFile("captcha.jpg")
			res, err = cli.SubmitCaptcha(text, res.CaptchaSign)
			continue
		case client.SMSNeededError:
			log.Warnf("账号已开启设备锁, 按 Enter 向手机 %v 发送短信验证码.", res.SMSPhone)
			readLine()
			if !cli.RequestSMS() {
				log.Warnf("发送验证码失败，可能是请求过于频繁.")
				return errors.WithStack(ErrSMSRequestError)
			}
			log.Warn("请输入短信验证码： (Enter 提交)")
			text = readLine()
			res, err = cli.SubmitSMS(text)
			continue
		case client.SMSOrVerifyNeededError:
			log.Warnf("账号已开启设备锁，请选择验证方式:")
			log.Warnf("1. 向手机 %v 发送短信验证码", res.SMSPhone)
			log.Warnf("2. 使用手机QQ扫码验证.")
			log.Warn("请输入(1 - 2)：")
			text = readIfTTY("2")
			if strings.Contains(text, "1") {
				if !cli.RequestSMS() {
					log.Warnf("发送验证码失败，可能是请求过于频繁.")
					return errors.WithStack(ErrSMSRequestError)
				}
				log.Warn("请输入短信验证码： (Enter 提交)")
				text = readLine()
				res, err = cli.SubmitSMS(text)
				continue
			}
			fallthrough
		case client.UnsafeDeviceError:
			log.Warnf("账号已开启设备锁，请前往 -> %v <- 验证后重启Bot.", res.VerifyUrl)
			log.Infof("按 Enter 或等待 5s 后继续....")
			readLineTimeout(time.Second * 5)
			os.Exit(0)
		case client.OtherLoginError, client.UnknownLoginError, client.TooManySMSRequestError:
			msg := res.ErrorMessage
			log.Warnf("登录失败: %v Code: %v", msg, res.Code)
			switch res.Code {
			case 235:
				log.Warnf("设备信息被封禁, 请删除 device.json 后重试.")
			case 237:
				log.Warnf("登录过于频繁, 请在手机QQ登录并根据提示完成认证后等一段时间重试")
			case 45:
				log.Warnf("你的账号被限制登录, 请配置 SignServer 后重试")
			}
			log.Infof("按 Enter 继续....")
			readLine()
			os.Exit(0)
		}
	}
}

func getTicket(u string) string {
	log.Warnf("请前往该地址验证 -> %v ", u)
	log.Warn("请输入ticket： (Enter 提交)")
	return readLine()
}
