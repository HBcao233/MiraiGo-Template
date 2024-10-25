package global

import (
	"bytes"
	"errors"
	"net/netip"
	"os"
	"strings"

	"github.com/Mrs4s/MiraiGo/utils"
	log "github.com/sirupsen/logrus"
)

const (
	// ImagePath go-cqhttp使用的图片缓存目录
	ImagePath = "data/images"
	// VoicePath go-cqhttp使用的语音缓存目录
	VoicePath = "data/voices"
	// VideoPath go-cqhttp使用的视频缓存目录
	VideoPath = "data/videos"
	// VersionsPath go-cqhttp使用的版本信息目录
	VersionsPath = "data/versions"
	// CachePath go-cqhttp使用的缓存目录
	CachePath = "data/cache"
	// DumpsPath go-cqhttp使用错误转储目录
	DumpsPath = "dumps"
	// HeaderAmr AMR文件头
	HeaderAmr = "#!AMR"
	// HeaderSilk Silkv3文件头
	HeaderSilk = "\x02#!SILK_V3"
)

// PathExists 判断给定path是否存在
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || errors.Is(err, os.ErrExist)
}

// ReadFile 读取文件
// 读取失败返回 nil
func ReadFile(path string) []byte {
	bytes, err := os.ReadFile(path)
	if err != nil {
		log.WithError(err).WithField("util", "ReadFile").Errorf("unable to read '%s'", path)
		return nil
	}
	return bytes
}

// ReadAllText 读取给定path对应文件，无法读取时返回空值
func ReadAllText(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Error(err)
		return ""
	}
	return string(b)
}

// WriteAllText 将给定text写入给定path
func WriteAllText(path, text string) error {
	return os.WriteFile(path, utils.S2B(text), 0o644)
}

// Check 检测err是否为nil
func Check(err error, deleteSession bool) {
	if err != nil {
		if deleteSession && PathExists("session.token") {
			_ = os.Remove("session.token")
		}
		log.Fatalf("遇到错误: %v", err)
	}
}

// IsAMRorSILK 判断给定文件是否为Amr或Silk格式
func IsAMRorSILK(b []byte) bool {
	return bytes.HasPrefix(b, []byte(HeaderAmr)) || bytes.HasPrefix(b, []byte(HeaderSilk))
}

// DelFile 删除一个给定path，并返回删除结果
func DelFile(path string) bool {
	err := os.Remove(path)
	if err != nil {
		// 删除失败
		log.Error(err)
		return false
	}
	// 删除成功
	log.Info(path + "删除成功")
	return true
}

// ReadAddrFile 从给定path中读取合法的IP地址与端口,每个IP地址以换行符"\n"作为分隔
func ReadAddrFile(path string) []netip.AddrPort {
	d, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	str := string(d)
	lines := strings.Split(str, "\n")
	var ret []netip.AddrPort
	for _, l := range lines {
		addr, err := netip.ParseAddrPort(l)
		if err == nil {
			ret = append(ret, addr)
		}
	}
	return ret
}
