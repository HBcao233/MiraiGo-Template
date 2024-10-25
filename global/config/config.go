package config

import (
	"os"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type LoginMethod string

const (
	LoginMethodToken  = "token"
	LoginMethodQRCode = "qrcode"
	LoginMethodCommon = "common"
)

// Account 账号配置
type Account struct {
	LoginMethod          LoginMethod  `yaml:"login-method"`
	Uin                  int64        `yaml:"uin"`
	Password             string       `yaml:"password"`
	Encrypt              bool         `yaml:"encrypt"`
	Status               int          `yaml:"status"`
	SignServers          []SignServer `yaml:"sign-servers"`
	RuleChangeSignServer int          `yaml:"rule-change-sign-server"`
	MaxCheckCount        uint         `yaml:"max-check-count"`
	SignServerTimeout    uint         `yaml:"sign-server-timeout"`
}

// SignServer 签名服务器
type SignServer struct {
	URL           string `yaml:"url"`
	Key           string `yaml:"key"`
	Authorization string `yaml:"authorization"`
}

type Config struct {
	Account *Account `yaml:"account"`
}

// Parse 从默认配置文件路径中获取
func Parse(path string) *Config {
	file, err := os.ReadFile(path)
	config := &Config{}
	if err == nil {
		err = yaml.NewDecoder(strings.NewReader(expand(string(file), os.Getenv))).Decode(config)
		if err != nil {
			log.Fatal("配置文件不合法!", err)
		}
	} else {
		os.Exit(0)
	}
	return config
}

// expand 使用正则进行环境变量展开
// os.ExpandEnv 字符 $ 无法逃逸
// https://github.com/golang/go/issues/43482
func expand(s string, mapping func(string) string) string {
	r := regexp.MustCompile(`\${([a-zA-Z_]+[a-zA-Z0-9_:/.]*)}`)
	return r.ReplaceAllStringFunc(s, func(s string) string {
		s = strings.Trim(s, "${}")
		before, after, ok := strings.Cut(s, ":")
		m := mapping(before)
		if ok && m == "" {
			return after
		}
		return m
	})
}
