package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/go-errors"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/patrickmn/go-cache"
	"github.com/yanjunhui/goini"
)

var (
	corpId         = "wwa7XXXXXXXXXXXX2"       //企业微信 corpid
	EncodingAESKey = "dm1XXXXXXXXXXXXXXXXXXXX" //EncodingAESKey

	TokenCache *cache.Cache
)

func init() {
	TokenCache = cache.New(6000*time.Second, 5*time.Second)
}

func main() {
	e := echo.New()
	e.Logger.SetLevel(log.INFO)
	e.Use(middleware.Logger())
	e.GET("/author", WxAuth)
	port := GetConfig.GetValue("http", "port")
	e.Logger.Fatal(e.Start(":8080"))
}

//发送信息
type Content struct {
	Content string `json:"content"`
}

//开启回调模式验证
func WxAuth(context echo.Context) error {

	echostr := context.FormValue("echostr")
	if echostr == "" {
		return errors.New("无法获取请求参数, echostr 为空")
	}

	wByte, err := base64.StdEncoding.DecodeString(echostr)
	if err != nil {
		return errors.New("接受微信请求参数 echostr base64解码失败(" + err.Error() + ")")
	}
	key, err := base64.StdEncoding.DecodeString(EncodingAESKey + "=")
	if err != nil {
		return errors.New("配置 EncodingAESKey base64解码失败(" + err.Error() + "), 请检查配置文件内 EncodingAESKey 是否和微信后台提供一致")
	}

	keyByte := []byte(key)
	x, err := AesDecrypt(wByte, keyByte)
	if err != nil {
		return errors.New("aes 解码失败(" + err.Error() + "), 请检查配置文件内 EncodingAESKey 是否和微信后台提供一致")
	}

	buf := bytes.NewBuffer(x[16:20])
	var length int32
	binary.Read(buf, binary.BigEndian, &length)

	//验证返回数据ID是否正确
	appIDstart := 20 + length
	if len(x) < int(appIDstart) {
		return errors.New("获取数据错误, 请检查 EncodingAESKey 配置")
	}
	id := x[appIDstart : int(appIDstart)+len(corpId)]
	if string(id) == corpId {
		return context.JSONBlob(200, x[20:20+length])
	}
	return errors.New("微信验证appID错误, 微信请求值: " + string(id) + ", 配置文件内配置为: " + corpId)
}

type AccessToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

//AES解密
func AesDecrypt(crypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Printf("aes解密失败: %v", err)
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = PKCS5UnPadding(origData)
	return origData, nil
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

//string 类型转 int
func StringToInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("agent 类型转换失败, 请检查配置文件中 agentid 配置是否为纯数字(%v)", err)
		return 0
	}
	return n
}

//json序列化(禁止 html 符号转义)
func encodeJson(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
