// Package donghao 冬浩验证系统 Go SDK
//
// 本SDK提供完整的冬浩验证系统API接口封装，支持：
//   - 用户管理（登录、注册、注销）
//   - 卡密管理（充值、绑定）
//   - 云变量/常量操作
//   - 云计算函数调用
//   - 黑名单管理
//   - 自动心跳维持在线状态
//
// 使用示例：
//
//	package main
//
//	import (
//	    "fmt"
//	    "log"
//	    "gitee.com/yuan71058/dong-hao-verification"
//	)
//
//	func main() {
//	    // 创建客户端（第二个参数是软件ID，从管理后台获取）
//	    client := donghao.NewClient("https://your-api-domain.com", 1)
//	    client.SetTimeout(30)
//
//	    // 用户登录
//	    result, err := client.Login("username", "password", "1.0", "mac123", "192.168.1.1", "client001")
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    if result.IsSuccess() {
//	        fmt.Println("登录成功，Token:", client.GetToken())
//	    } else {
//	        fmt.Println("登录失败:", result.Msg)
//	    }
//	}
//
// API调用方式:
//   - 所有请求发送到统一入口: {BaseURL}/api.php?appid={AppID}
//   - 通过action参数指定接口名称（如：login, logout, heartbeat等）
//
// 版本: 1.0
// 作者: 冬浩验证系统
// 日期: 2026-03-31
package donghao

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	// chacha20poly1305 "golang.org/x/crypto/chacha20poly1305" // 已禁用: OpenSSL 1.1.1 不支持 AEAD Tag
)

// Client 冬浩验证客户端结构体
//
// 提供完整的API接口封装，包括用户管理、卡密管理、云变量操作等功能
type Client struct {
	BaseURL           string        // API服务器基础URL
	AppID             int           // 软件ID（必填）
	AppKey            string        // 应用密钥（用于签名）
	SignTemplate      string        // 签名模板（如"[data]xxx[key]yyy"）
	NeedSign          bool          // 是否需要签名
	UseGBK            bool          // RC4加密是否使用GBK编码（与PHP兼容）
	Timeout           time.Duration // HTTP请求超时时间
	EncryptionType    int           // 加密类型
	EncryptionKey     string        // 加密密钥（RC4/RSA/自定义加密时使用）
	AesGcmKey         string        // AES-256-GCM密钥（ENC_AES_GCM模式使用，32字节）
	// ChaChaKey         string        // ChaCha20-Poly1305密钥 - 已禁用
	HeartbeatInterval time.Duration // 心跳间隔
	currentToken      string        // 当前登录token
	currentUUID       string        // 当前客户端UUID
	currentUser       string        // 当前登录用户名
	heartbeatRunning  bool          // 心跳运行标志
	heartbeatCancel   chan bool     // 心跳取消通道
	heartbeatMutex    sync.Mutex    // 心跳互斥锁
	httpClient        *http.Client  // HTTP客户端
}

// Result API返回结果结构体
//
// 封装API调用的返回结果，包含状态码、消息和数据
type Result struct {
	Code   int         `json:"code"`   // 返回状态码：1=成功，其他=失败
	Result interface{} `json:"result"` // 返回结果数据
	Data   interface{} `json:"data"`   // 返回数据内容（兼容旧版）
	Uuid   string      `json:"uuid"`   // 客户端UUID
	Token  string      `json:"token"`  // 新Token
	T      int64       `json:"t"`      // 时间戳
}

// Msg 获取返回消息
func (r *Result) Msg() string {
	if r.Result == nil {
		return ""
	}
	switch v := r.Result.(type) {
	case string:
		return v
	case map[string]interface{}:
		if retInfo, ok := v["ret_info"].(string); ok {
			return retInfo
		}
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetTokenID 获取登录返回的tokenid
func (r *Result) GetTokenID() string {
	if r.Result == nil {
		return ""
	}
	switch v := r.Result.(type) {
	case map[string]interface{}:
		if tokenID, ok := v["tokenid"].(string); ok {
			return tokenID
		}
		if tokenID, ok := v["tokenid"].(float64); ok {
			return strconv.FormatFloat(tokenID, 'f', 0, 64)
		}
		return ""
	default:
		return ""
	}
}

// GetResultMap 获取result字段作为map
func (r *Result) GetResultMap() map[string]interface{} {
	if r.Result == nil {
		return nil
	}
	if m, ok := r.Result.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// GetVariableValue 获取云变量值（自动base64解码）
//
// 返回:
//   - string: 解码后的变量值
//   - error: 解码错误
//
// 说明:
//
//	PHP返回的云变量值是base64编码的，此方法自动解码
//	适用于: getvariable, constant 接口
func (r *Result) GetVariableValue() (string, error) {
	m := r.GetResultMap()
	if m == nil {
		return "", fmt.Errorf("result不是map类型")
	}

	valuesRaw, ok := m["values"]
	if !ok {
		return "", fmt.Errorf("未找到values字段")
	}

	valuesStr, ok := valuesRaw.(string)
	if !ok {
		return "", fmt.Errorf("values不是字符串类型")
	}

	decoded, err := base64.StdEncoding.DecodeString(valuesStr)
	if err != nil {
		return "", fmt.Errorf("base64解码失败: %v", err)
	}

	return string(decoded), nil
}

// GetData 获取用户数据（自动base64解码）
//
// 返回:
//   - string: 解码后的用户数据
//   - error: 解码错误
//
// 说明:
//
//	PHP返回的用户数据(data字段)是base64编码的，此方法自动解码
//	适用于: login, getuser 接口
func (r *Result) GetData() (string, error) {
	m := r.GetResultMap()
	if m == nil {
		return "", fmt.Errorf("result不是map类型")
	}

	dataRaw, ok := m["data"]
	if !ok {
		return "", fmt.Errorf("未找到data字段")
	}

	dataStr, ok := dataRaw.(string)
	if !ok {
		return "", fmt.Errorf("data不是字符串类型")
	}

	decoded, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return "", fmt.Errorf("base64解码失败: %v", err)
	}

	return string(decoded), nil
}

// GetGroupData 获取用户组数据（自动base64解码）
//
// 返回:
//   - string: 解码后的用户组数据
//   - error: 解码错误
//
// 说明:
//
//	PHP返回的用户组数据(groupdata字段)是base64编码的，此方法自动解码
//	适用于: login, getuser 接口
func (r *Result) GetGroupData() (string, error) {
	m := r.GetResultMap()
	if m == nil {
		return "", fmt.Errorf("result不是map类型")
	}

	groupDataRaw, ok := m["groupdata"]
	if !ok {
		return "", fmt.Errorf("未找到groupdata字段")
	}

	groupDataStr, ok := groupDataRaw.(string)
	if !ok {
		return "", fmt.Errorf("groupdata不是字符串类型")
	}

	decoded, err := base64.StdEncoding.DecodeString(groupDataStr)
	if err != nil {
		return "", fmt.Errorf("base64解码失败: %v", err)
	}

	return string(decoded), nil
}

// 加密类型常量
const (
	ENC_NONE    = 0 // 不加密，明文传输
	ENC_RC4     = 1 // RC4流加密算法
	ENC_RSA     = 2 // RSA非对称加密
	ENC_BASE64  = 3 // Base64编码（仅编码，非加密）
	ENC_CUSTOM  = 4 // 自定义加密算法
	ENC_AES_GCM = 5 // AES-256-GCM认证加密（与PHP mi_type==5兼容）
	// ENC_CHACHA  = 6 // ChaCha20-Poly1305 - 已禁用: OpenSSL 1.1.1 不支持 AEAD Tag
)

// NewClient 创建新的冬浩验证客户端
//
// 参数:
//   - baseURL: API服务器基础URL，例如：https://your-domain.com
//   - appID: 软件ID，从管理后台获取
//
// 返回:
//   - *Client: 客户端实例
//
// 示例:
//
//	client := donghao.NewClient("https://api.example.com", 1)
//	client.SetTimeout(30)
func NewClient(baseURL string, appID int) *Client {
	return &Client{
		BaseURL:           strings.TrimSuffix(baseURL, "/"),
		AppID:             appID,
		Timeout:           30 * time.Second,
		EncryptionType:    ENC_NONE,
		UseGBK:            true,
		HeartbeatInterval: 150 * time.Second,
		heartbeatCancel:   make(chan bool),
		httpClient:        &http.Client{Timeout: 30 * time.Second},
	}
}

// SetTimeout 设置HTTP请求超时时间
//
// 参数:
//   - seconds: 超时秒数
func (c *Client) SetTimeout(seconds int) {
	c.Timeout = time.Duration(seconds) * time.Second
	c.httpClient.Timeout = c.Timeout
}

// SetEncryption 设置数据加密方式
//
// 参数:
//   - encType: 加密类型，参见 ENC_NONE, ENC_RC4, ENC_RSA, ENC_BASE64, ENC_CUSTOM, ENC_AES_GCM
//   - key: 加密密钥（RC4/自定义加密/AES-GCM时使用）
func (c *Client) SetEncryption(encType int, key string) {
	c.EncryptionType = encType
	c.EncryptionKey = key
	switch encType {
	case ENC_AES_GCM:
		c.AesGcmKey = key
	// case ENC_CHACHA: // 已禁用
	// 	c.ChaChaKey = key
	}
}

// SetSignConfig 设置签名配置
//
// 参数:
//   - appKey: 应用密钥
//   - template: 签名模板（如"[data]xxx[key]yyy"）
//   - needSign: 是否需要签名
func (c *Client) SetSignConfig(appKey string, template string, needSign bool) {
	c.AppKey = appKey
	c.SignTemplate = template
	c.NeedSign = needSign
}

// SetHeartbeatInterval 设置自动心跳间隔
//
// 参数:
//   - seconds: 心跳间隔秒数，默认300秒（5分钟）
//
// 心跳用于维持用户在线状态，防止会话过期
func (c *Client) SetHeartbeatInterval(seconds int) {
	c.HeartbeatInterval = time.Duration(seconds) * time.Second
}

// GetToken 获取当前登录Token
//
// 返回:
//   - string: token值，未登录时返回空字符串
func (c *Client) GetToken() string {
	return c.currentToken
}

// GetUUID 获取当前客户端UUID
//
// 返回:
//   - string: UUID值，未登录时返回空字符串
func (c *Client) GetUUID() string {
	return c.currentUUID
}

// GetCurrentUser 获取当前登录用户名
//
// 返回:
//   - string: 用户名，未登录时返回空字符串
func (c *Client) GetCurrentUser() string {
	return c.currentUser
}

// httpPost 发送HTTP POST请求（内部方法）
//
// 参数:
//   - action: API接口名称（如：login, logout, heartbeat等）
//   - params: POST请求参数
//
// 返回:
//   - *Result: 解析后的结果
//   - error: 请求错误
func (c *Client) httpPost(action string, params map[string]string) (*Result, error) {
	apiURL := fmt.Sprintf("%s/api.php?appid=%d", c.BaseURL, c.AppID)

	params["action"] = action

	// 自动注入必需参数：t（时间戳）、uuid、token
	params["t"] = fmt.Sprintf("%d", time.Now().Unix())
	if c.currentUUID != "" {
		params["uuid"] = c.currentUUID
	}
	if c.currentToken != "" {
		params["token"] = c.currentToken
	}

	var postData string
	var plainData string
	var signStr string

	if c.EncryptionType == ENC_NONE {
		data := url.Values{}
		for key, value := range params {
			data.Set(key, value)
		}
		postData = data.Encode()
		plainData = postData

		if c.NeedSign && c.AppKey != "" {
			signStr = c.generateSignForPlain(params)
		}
	} else {
		data := url.Values{}
		for key, value := range params {
			data.Set(key, value)
		}
		plainData = data.Encode()

		encryptedData, err := c.Encrypt(plainData)
		if err != nil {
			return nil, fmt.Errorf("加密失败: %v", err)
		}

		postData = "data=" + url.QueryEscape(encryptedData)

		if c.NeedSign && c.AppKey != "" {
			signStr = c.generateSignForEncrypted(encryptedData)
		}
	}

	if signStr != "" {
		postData = postData + "&sign=" + signStr
	}

	fmt.Printf("[DEBUG] HTTP POST请求: %s\n", apiURL)
	if c.EncryptionType == ENC_NONE {
		fmt.Printf("[DEBUG] 提交数据(明文): %s\n", postData)
	} else {
		fmt.Printf("[DEBUG] 提交数据(加密前): %s\n", plainData)
		fmt.Printf("[DEBUG] 提交数据(加密后): %s\n", postData)
	}

	resp, err := c.httpClient.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(postData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[DEBUG] 响应数据: %s\n", string(body))

	if c.EncryptionType != ENC_NONE {
		var encryptedResp struct {
			Data string `json:"data"`
			Sign string `json:"sign"`
		}
		if err := json.Unmarshal(body, &encryptedResp); err != nil {
			return nil, err
		}

		decryptedData, err := c.Decrypt(encryptedResp.Data)
		if err != nil {
			return nil, fmt.Errorf("解密失败: %v", err)
		}

		fmt.Printf("[DEBUG] 解密后数据: %s\n", decryptedData)

		var result Result
		if err := json.Unmarshal([]byte(decryptedData), &result); err != nil {
			return nil, err
		}
		// Token轮换机制：更新为服务端返回的新token和uuid
		if result.Token != "" {
			c.currentToken = result.Token
		}
		if result.Uuid != "" {
			c.currentUUID = result.Uuid
		}
		return &result, nil
	}

	var plainResp struct {
		Data *Result `json:"data"`
		Sign string  `json:"sign"`
	}
	if err := json.Unmarshal(body, &plainResp); err != nil {
		return nil, err
	}

	if plainResp.Data != nil {
		if plainResp.Data.Token != "" {
			c.currentToken = plainResp.Data.Token
		}
		if plainResp.Data.Uuid != "" {
			c.currentUUID = plainResp.Data.Uuid
		}
		return plainResp.Data, nil
	}

	var result Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// Token轮换机制：更新为服务端返回的新token和uuid
	if result.Token != "" {
		c.currentToken = result.Token
	}
	if result.Uuid != "" {
		c.currentUUID = result.Uuid
	}

	return &result, nil
}

// generateSignForPlain 生成明文模式的签名
func (c *Client) generateSignForPlain(params map[string]string) string {
	var signParts []string
	for k, v := range params {
		if k != "sign" && k != "act" && k != "appid" {
			signParts = append(signParts, fmt.Sprintf("%s=%s", k, v))
		}
	}
	signData := strings.Join(signParts, "&")

	if c.SignTemplate != "" {
		signData = strings.Replace(c.SignTemplate, "[data]", signData, 1)
		signData = strings.Replace(signData, "[key]", c.AppKey, 1)
	} else {
		signData = signData + c.AppKey
	}

	return GenerateMD5(signData)
}

// generateSignForEncrypted 生成加密模式的签名
func (c *Client) generateSignForEncrypted(data string) string {
	signData := data
	if c.SignTemplate != "" {
		signData = strings.Replace(c.SignTemplate, "[data]", data, 1)
		signData = strings.Replace(signData, "[key]", c.AppKey, 1)
	} else {
		signData = data + c.AppKey
	}
	return GenerateMD5(signData)
}

// Login 用户登录
//
// 参数:
//   - user: 用户名
//   - pwd: 密码（明文或MD5，取决于服务器配置）
//   - ver: 软件版本号
//   - mac: 机器码（唯一标识设备）
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 登录结果，成功时Data字段包含token
//   - error: 请求错误
//
// 说明:
//
//	登录成功后会自动保存token，用于后续需要登录的接口
//
// 示例:
//
//	result, err := client.Login("username", "password", "1.0", "mac123", "192.168.1.1", "client001")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if result.IsSuccess() {
//	    fmt.Println("登录成功，Token:", client.GetToken())
//	}
func (c *Client) Login(user, pwd, ver, mac, ip, clientid string) (*Result, error) {
	// 生成初始UUID（首次登录时使用）
	if c.currentUUID == "" {
		c.currentUUID = generateUUID()
	}

	params := map[string]string{
		"user":     user,
		"pwd":      pwd,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	result, err := c.httpPost("login", params)
	if err != nil {
		return nil, err
	}

	if result.IsSuccess() {
		if tokenID := result.GetTokenID(); tokenID != "" {
			c.currentToken = tokenID
		} else if result.Token != "" {
			c.currentToken = result.Token
		}
		c.currentUser = user
	}

	return result, nil
}

// LoginCard 卡密登录
//
// 使用卡密直接登录，无需用户名和密码
// 注意: 此功能需要后端开启"充值卡登录模式"(dl_type=1)
//
// 参数:
//   - card: 卡密
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 登录结果，包含token和用户信息
//   - error: 请求错误
//
// 适用场景:
//   - 发卡平台销售卡密
//   - 用户直接使用卡密登录软件
//
// 示例:
//
//	result, err := client.LoginCard("CARD-123456", "1.0", "mac123", "192.168.1.1", "client001")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if result.IsSuccess() {
//	    fmt.Println("卡密登录成功，Token:", client.GetToken())
//	}
func (c *Client) LoginCard(card, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     card,
		"pwd":      "",
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	result, err := c.httpPost("login", params)
	if err != nil {
		return nil, err
	}

	if result.IsSuccess() {
		if tokenID := result.GetTokenID(); tokenID != "" {
			c.currentToken = tokenID
		} else if result.Token != "" {
			c.currentToken = result.Token
		}
		c.currentUser = card
	}

	return result, nil
}

// Reg 用户注册
//
// 参数:
//   - user: 用户名（唯一）
//   - pwd: 密码
//   - card: 卡密（可选，用于直接充值）
//   - userqq: 用户QQ号
//   - email: 用户邮箱
//   - tjr: 推荐人用户名（可选）
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 注册结果
//   - error: 请求错误
func (c *Client) Reg(user, pwd, card, userqq, email, tjr, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"pwd":      pwd,
		"card":     card,
		"userqq":   userqq,
		"email":    email,
		"tjr":      tjr,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("reg", params)
}

// Logout 用户退出登录
//
// 参数:
//   - user: 用户名
//   - tokenid: 登录token
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 注销结果
//   - error: 请求错误
//
// 说明:
//
//	注销成功后会清除本地保存的token
func (c *Client) Logout(user, tokenid, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"tokenid":  tokenid,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	result, err := c.httpPost("logout", params)
	if err != nil {
		return nil, err
	}

	if result.IsSuccess() {
		c.currentToken = ""
		c.currentUser = ""
	}

	return result, nil
}

// Heartbeat 发送心跳包
//
// 参数:
//   - user: 用户名
//   - tokenid: 登录token
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 心跳结果
//   - error: 请求错误
//
// 说明:
//
//	心跳用于维持用户在线状态，建议每5分钟发送一次
func (c *Client) Heartbeat(user, tokenid, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"tokenid":  tokenid,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("heartbeat", params)
}

// GetUser 获取用户信息
//
// 参数:
//   - user: 用户名
//   - tokenid: 登录token
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 用户信息，Data字段包含用户详细信息JSON
//   - error: 请求错误
func (c *Client) GetUser(user, tokenid, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"tokenid":  tokenid,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("getuser", params)
}

// GetUdata 获取用户云端数据
//
// 参数:
//   - user: 用户名
//   - tokenid: 登录token
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 用户数据
//   - error: 请求错误
func (c *Client) GetUdata(user, tokenid, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"tokenid":  tokenid,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("getudata", params)
}

// SetUdata 设置用户云端数据
//
// 参数:
//   - user: 用户名
//   - tokenid: 登录token
//   - udata: 用户数据（字符串格式，建议JSON）
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 设置结果
//   - error: 请求错误
func (c *Client) SetUdata(user, tokenid, udata, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"tokenid":  tokenid,
		"udata":    udata,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("setudata", params)
}

// Uppwd 修改密码
//
// 参数:
//   - user: 用户名
//   - pwd: 原密码
//   - newpwd: 新密码
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 修改结果
//   - error: 请求错误
//
// 说明:
//
//	此接口不需要tokenid，使用密码验证身份
func (c *Client) Uppwd(user, pwd, newpwd, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"pwd":      pwd,
		"newpwd":   newpwd,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("uppwd", params)
}

// Binding 绑定换绑（修改用户信息）
//
// 参数:
//   - user: 用户名
//   - pwd: 密码
//   - newuser: 新用户名（可选，为空则不修改）
//   - newmac: 新机器码（可选，为空则不修改）
//   - newip: 新IP地址（可选，为空则不修改）
//   - newuserqq: 新QQ号（可选，为空则不修改）
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 绑定结果
//   - error: 请求错误
//
// 说明:
//
//	用于更换用户的授权信息（用户名、机器码、IP、QQ），不需要tokenid
func (c *Client) Binding(user, pwd, newuser, newmac, newip, newuserqq, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":      user,
		"pwd":       pwd,
		"newuser":   newuser,
		"newmac":    newmac,
		"newip":     newip,
		"newuserqq": newuserqq,
		"ver":       ver,
		"mac":       mac,
		"ip":        ip,
		"clientid":  clientid,
	}

	return c.httpPost("binding", params)
}

// Bindreferrer 绑定推荐人
//
// 参数:
//   - user: 用户名
//   - pwd: 密码
//   - tjr: 推荐人用户名
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 绑定结果
//   - error: 请求错误
//
// 说明:
//
//	不需要tokenid，使用密码验证身份
func (c *Client) Bindreferrer(user, pwd, tjr, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"pwd":      pwd,
		"tjr":      tjr,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("bindreferrer", params)
}

// Recharge 卡密充值
//
// 参数:
//   - user: 用户名
//   - card: 卡密
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 充值结果
//   - error: 请求错误
//
// 说明:
//
//	使用卡密为用户账户充值时间或点数，不需要tokenid
func (c *Client) Recharge(user, card, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"card":     card,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("recharge", params)
}

// GetVariable 获取云变量
//
// 参数:
//   - user: 用户名
//   - tokenid: 登录token
//   - cloudkey: 变量名
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 变量值
//   - error: 请求错误
//
// 说明:
//
//	云变量是用户级别的键值对存储
func (c *Client) GetVariable(user, tokenid, cloudkey, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"tokenid":  tokenid,
		"cloudkey": cloudkey,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("getvariable", params)
}

// SetVariable 设置云变量
//
// 参数:
//   - user: 用户名
//   - tokenid: 登录token
//   - cloudkey: 变量名
//   - cloudvalue: 变量值
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 设置结果
//   - error: 请求错误
func (c *Client) SetVariable(user, tokenid, cloudkey, cloudvalue, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":       user,
		"tokenid":    tokenid,
		"cloudkey":   cloudkey,
		"cloudvalue": cloudvalue,
		"ver":        ver,
		"mac":        mac,
		"ip":         ip,
		"clientid":   clientid,
	}

	return c.httpPost("setvariable", params)
}

// DelVariable 删除云变量
//
// 参数:
//   - user: 用户名
//   - tokenid: 登录token
//   - cloudkey: 变量名
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 删除结果
//   - error: 请求错误
func (c *Client) DelVariable(user, tokenid, cloudkey, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"tokenid":  tokenid,
		"cloudkey": cloudkey,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("delvariable", params)
}

// Constant 获取云端常量
//
// 参数:
//   - user: 用户名
//   - tokenid: 登录token（可选，取决于服务器配置）
//   - cloudkey: 常量名
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 常量值
//   - error: 请求错误
//
// 说明:
//
//	常量是软件级别的只读配置，所有用户共享
func (c *Client) Constant(user, tokenid, cloudkey, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"tokenid":  tokenid,
		"cloudkey": cloudkey,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("constant", params)
}

// Func 调用云计算函数（需登录）
//
// 参数:
//   - user: 用户名
//   - tokenid: 登录token
//   - fun: 函数名
//   - para: 函数参数（JSON格式）
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 函数执行结果
//   - error: 请求错误
//
// 说明:
//
//	调用服务器端定义的PHP函数
func (c *Client) Func(user, tokenid, fun, para, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"tokenid":  tokenid,
		"fun":      fun,
		"para":     para,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("func", params)
}

// Func2 调用云计算函数2（无需登录）
//
// 参数:
//   - fun: 函数名
//   - para: 函数参数（JSON格式）
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 函数执行结果
//   - error: 请求错误
//
// 说明:
//
//	无需登录即可调用的公共函数
func (c *Client) Func2(fun, para, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"fun":      fun,
		"para":     para,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("func2", params)
}

// CallPHP 调用PHP函数（需登录）
//
// 参数:
//   - user: 用户名
//   - tokenid: 登录token
//   - fun: 函数名
//   - para: 函数参数（JSON格式）
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 函数执行结果
//   - error: 请求错误
func (c *Client) CallPHP(user, tokenid, fun, para, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"tokenid":  tokenid,
		"fun":      fun,
		"para":     para,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("callphp", params)
}

// CallPHP2 调用PHP函数2（无需登录）
//
// 参数:
//   - fun: 函数名
//   - para: 函数参数（JSON格式）
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 函数执行结果
//   - error: 请求错误
func (c *Client) CallPHP2(fun, para, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"fun":      fun,
		"para":     para,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("callphp2", params)
}

// GetBlack 查询黑名单
//
// 参数:
//   - bType: 黑名单类型（如：ip, mac, user）
//   - bData: 黑名单数据（具体的IP、MAC或用户名）
//
// 返回:
//   - *Result: 查询结果
//   - error: 请求错误
//
// 说明:
//
//	检查指定的数据是否在黑名单中，不需要ver, mac, ip, clientid参数
func (c *Client) GetBlack(bType, bData string) (*Result, error) {
	params := map[string]string{
		"b_type": bType,
		"b_data": bData,
	}

	return c.httpPost("getblack", params)
}

// SetBlack 添加黑名单
//
// 参数:
//   - bType: 黑名单类型
//   - bData: 黑名单数据
//   - bBz: 备注说明
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 添加结果
//   - error: 请求错误
func (c *Client) SetBlack(bType, bData, bBz, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"b_type":   bType,
		"b_data":   bData,
		"b_bz":     bBz,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("setblack", params)
}

// CheckAuth 验证账号密码
//
// 参数:
//   - user: 用户名
//   - pwd: 密码
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//   - md5: MD5校验值（可选，取决于服务器配置）
//
// 返回:
//   - *Result: 验证结果
//   - error: 请求错误
//
// 说明:
//
//	验证用户账号密码是否正确，不生成登录token
func (c *Client) CheckAuth(user, pwd, ver, mac, ip, clientid, md5 string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"pwd":      pwd,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
		"md5":      md5,
	}

	return c.httpPost("checkauth", params)
}

// DeductPoints 扣除积分
//
// 参数:
//   - user: 用户名
//   - tokenid: 登录token
//   - sl: 扣除数量
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 扣除结果
//   - error: 请求错误
func (c *Client) DeductPoints(user, tokenid string, sl int, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"tokenid":  tokenid,
		"sl":       strconv.Itoa(sl),
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("deductpoints", params)
}

// AddLog 添加日志记录
//
// 参数:
//   - user: 用户名
//   - infos: 日志信息
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 记录结果
//   - error: 请求错误
//
// 说明:
//
//	记录用户操作日志到服务器，不需要tokenid
func (c *Client) AddLog(user, infos, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"infos":    infos,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("addlog", params)
}

// Init 获取软件初始化信息
//
// 参数:
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 软件信息
//   - error: 请求错误
//
// 说明:
//
//	获取软件基本信息，包括名称、公告、静态数据等
//	返回的data字段是base64编码的，使用GetData()方法解码
func (c *Client) Init(ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("init", params)
}

// Notice 获取公告信息
//
// 参数:
//   - title: 公告标题（可选，为空则获取所有公告）
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 公告列表
//   - error: 请求错误
//
// 说明:
//
//	获取软件公告信息，可指定标题获取特定公告
//	返回data_list数组包含公告列表
func (c *Client) Notice(title, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"title":    title,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("notice", params)
}

// Ver 获取版本信息
//
// 参数:
//   - ver: 当前软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 版本信息
//   - error: 请求错误
//
// 说明:
//
//	检查是否有新版本，返回新版本信息
//	返回update_text字段是base64解码后的内容
func (c *Client) Ver(ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("ver", params)
}

// GetUdata2 获取用户云数据2
//
// 参数:
//   - user: 用户名
//   - tokenid: 登录token
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 用户云数据
//   - error: 请求错误
//
// 说明:
//
//	获取用户的第二块云数据区域（data3字段）
func (c *Client) GetUdata2(user, tokenid, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"tokenid":  tokenid,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("getudata2", params)
}

// SetUdata2 设置用户云数据2
//
// 参数:
//   - user: 用户名
//   - tokenid: 登录token
//   - udata: 用户数据
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 操作结果
//   - error: 请求错误
//
// 说明:
//
//	设置用户的第二块云数据区域（data3字段）
func (c *Client) SetUdata2(user, tokenid, udata, ver, mac, ip, clientid string) (*Result, error) {
	params := map[string]string{
		"user":     user,
		"tokenid":  tokenid,
		"udata":    udata,
		"ver":      ver,
		"mac":      mac,
		"ip":       ip,
		"clientid": clientid,
	}

	return c.httpPost("setudata2", params)
}

// Relay 中转请求
//
// 参数:
//   - params: 要中转的参数map
//   - ver: 软件版本号
//   - mac: 机器码
//   - ip: 客户端IP地址
//   - clientid: 客户端ID
//
// 返回:
//   - *Result: 中转返回的数据
//   - error: 请求错误
//
// 说明:
//
//	将请求中转到配置的目标URL，用于跨服务器通信
func (c *Client) Relay(params map[string]string, ver, mac, ip, clientid string) (*Result, error) {
	params["ver"] = ver
	params["mac"] = mac
	params["ip"] = ip
	params["clientid"] = clientid

	return c.httpPost("relay", params)
}

// ==================== 自动心跳功能 - 已禁用 ====================

// StartAutoHeartbeat 启动自动心跳 - 已禁用
//
// func (c *Client) StartAutoHeartbeat(user, tokenid, ver, mac, ip, clientid string) {
// 	c.heartbeatMutex.Lock()
// 	defer c.heartbeatMutex.Unlock()
//
// 	if c.heartbeatRunning {
// 		c.StopAutoHeartbeat()
// 	}
//
// 	c.heartbeatRunning = true
// 	go c.heartbeatLoop(user, tokenid, ver, mac, ip, clientid)
// }

// StopAutoHeartbeat 停止自动心跳 - 已禁用
//
// func (c *Client) StopAutoHeartbeat() {
// 	c.heartbeatMutex.Lock()
// 	defer c.heartbeatMutex.Unlock()
//
// 	if c.heartbeatRunning {
// 		c.heartbeatRunning = false
// 		close(c.heartbeatCancel)
// 		c.heartbeatCancel = make(chan bool)
// 	}
// }

// heartbeatLoop 心跳循环（内部方法）- 已禁用
//
// func (c *Client) heartbeatLoop(user, tokenid, ver, mac, ip, clientid string) {
// 	ticker := time.NewTicker(c.HeartbeatInterval)
// 	defer ticker.Stop()
//
// 	for {
// 		select {
// 		case <-ticker.C:
// 			c.Heartbeat(user, tokenid, ver, mac, ip, clientid)
// 		case <-c.heartbeatCancel:
// 			return
// 		}
// 	}
// }

// IsSuccess 检查是否成功
//
// 返回:
//   - bool: true表示成功（Code == 200 或 Code == 1）
func (r *Result) IsSuccess() bool {
	return r.Code == 200 || r.Code == 1
}

// GetDataString 获取数据字符串
//
// 返回:
//   - string: Data字段的字符串值
func (r *Result) GetDataString() string {
	if r.Data == nil {
		return ""
	}
	switch v := r.Data.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetDataInt 获取数据整数
//
// 返回:
//   - int: Data字段的整数值
func (r *Result) GetDataInt() int {
	if r.Data == nil {
		return 0
	}
	switch v := r.Data.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		i, _ := strconv.Atoi(v)
		return i
	default:
		return 0
	}
}

// Encrypt 加密数据
//
// 参数:
//   - data: 待加密数据
//
// 返回:
//   - string: 加密后的数据
//   - error: 加密错误
func (c *Client) Encrypt(data string) (string, error) {
	switch c.EncryptionType {
	case ENC_NONE:
		return data, nil
	case ENC_RC4:
		return RC4CryptWithEncoding(data, c.EncryptionKey, false, c.UseGBK)
	case ENC_RSA:
		return RSAPrivateEncrypt(data, c.EncryptionKey)
	case ENC_BASE64:
		return base64.StdEncoding.EncodeToString([]byte(data)), nil
	case ENC_CUSTOM:
		return data, nil
	case ENC_AES_GCM:
		return aesGcmEncrypt(data, c.AesGcmKey)
	// case ENC_CHACHA: // 已禁用
	// 	return chachaEncrypt(data, c.ChaChaKey)
	default:
		return "", errors.New("不支持的加密类型")
	}
}

// Decrypt 解密数据
//
// 参数:
//   - data: 待解密数据
//
// 返回:
//   - string: 解密后的数据
//   - error: 解密错误
func (c *Client) Decrypt(data string) (string, error) {
	switch c.EncryptionType {
	case ENC_NONE:
		return data, nil
	case ENC_RC4:
		return RC4CryptWithEncoding(data, c.EncryptionKey, true, c.UseGBK)
	case ENC_RSA:
		return RSAPrivateDecrypt(data, c.EncryptionKey)
	case ENC_BASE64:
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	case ENC_CUSTOM:
		return data, nil
	case ENC_AES_GCM:
		return aesGcmDecrypt(data, c.AesGcmKey)
	// case ENC_CHACHA: // 已禁用
	// 	return chachaDecrypt(data, c.ChaChaKey)
	default:
		return "", errors.New("不支持的加密类型")
	}
}

// GenerateMD5 生成字符串的MD5哈希值
//
// 参数:
//   - text: 要计算哈希的字符串
//
// 返回:
//   - string: MD5哈希值的十六进制字符串
func GenerateMD5(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// generateUUID 生成UUID v4
//
// 返回:
//   - string: UUID字符串（如：550e8400-e29b-41d4-a716-446655440000）
func generateUUID() string {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		now := time.Now().UnixNano()
		return fmt.Sprintf("%08x-%04x-4%03x-%04x-%012x",
			now&0xFFFFFFFF,
			(now>>32)&0xFFFF,
			(now>>48)&0x0FFF,
			(now>>60)&0xFFFF,
			now&0xFFFFFFFFFFFF)
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// GenerateSign 根据参数生成签名
//
// 参数:
//   - params: 请求参数map
//   - appKey: 应用密钥
//   - template: 签名模板（如"[data]123[key]456"）
//
// 返回:
//   - string: 生成的签名字符串
func GenerateSign(params map[string]string, appKey string, template string) string {
	var signParts []string
	for k, v := range params {
		if k != "sign" && k != "act" && k != "appid" {
			signParts = append(signParts, fmt.Sprintf("%s=%s", k, v))
		}
	}
	signStr := strings.Join(signParts, "&")

	if template != "" {
		signStr = strings.Replace(template, "[data]", signStr, 1)
		signStr = strings.Replace(signStr, "[key]", appKey, 1)
	} else {
		signStr = signStr + appKey
	}

	return GenerateMD5(signStr)
}

// GenerateSignForData 根据数据生成签名
//
// 参数:
//   - data: 要签名的数据字符串
//   - appKey: 应用密钥
//   - template: 签名模板（如"[data]123[key]456"）
//
// 返回:
//   - string: 生成的签名字符串
func GenerateSignForData(data string, appKey string, template string) string {
	signStr := template
	if template != "" {
		signStr = strings.Replace(template, "[data]", data, 1)
		signStr = strings.Replace(signStr, "[key]", appKey, 1)
	} else {
		signStr = data + appKey
	}
	return GenerateMD5(signStr)
}

// RC4Crypt RC4加密/解密函数
//
// 参数:
//   - data: 要加密或解密的数据（解密时为十六进制字符串）
//   - key: RC4密钥
//   - decrypt: 是否为解密操作（true=解密，false=加密）
//
// 返回:
//   - string: 加密后的十六进制字符串（加密时）或解密后的原始数据（解密时）
//   - error: 操作失败时的错误信息
func RC4Crypt(data string, key string, decrypt bool) (string, error) {
	return RC4CryptWithEncoding(data, key, decrypt, false)
}

// RC4CryptWithEncoding RC4加密/解密函数（支持GBK编码）
//
// 参数:
//   - data: 要加密或解密的数据（解密时为十六进制字符串）
//   - key: RC4密钥
//   - decrypt: 是否为解密操作（true=解密，false=加密）
//   - useGBK: 是否使用GBK编码（与PHP兼容）
//
// 返回:
//   - string: 加密后的十六进制字符串（加密时）或解密后的原始数据（解密时）
//   - error: 操作失败时的错误信息
func RC4CryptWithEncoding(data string, key string, decrypt bool, useGBK bool) (string, error) {
	var dataBytes []byte
	var keyBytes []byte

	if useGBK {
		dataBytes = utf8ToGBK(data)
		keyBytes = utf8ToGBK(key)
	} else {
		dataBytes = []byte(data)
		keyBytes = []byte(key)
	}

	if decrypt {
		decoded, err := hex.DecodeString(data)
		if err != nil {
			return "", fmt.Errorf("hex decode failed: %v", err)
		}
		dataBytes = decoded
	}

	keyLen := len(keyBytes)
	if keyLen == 0 {
		return "", fmt.Errorf("key cannot be empty")
	}

	dataLen := len(dataBytes)

	var s [256]byte
	var k [256]byte

	for i := 0; i < 256; i++ {
		s[i] = byte(i)
		k[i] = keyBytes[i%keyLen]
	}

	var j int = 0
	for i := 0; i < 256; i++ {
		j = (j + int(s[i]) + int(k[i])) % 256
		s[i], s[j] = s[j], s[i]
	}

	var result []byte
	var i2, j2 int = 0, 0
	for n := 0; n < dataLen; n++ {
		i2 = (i2 + 1) % 256
		j2 = (j2 + int(s[i2])) % 256
		s[i2], s[j2] = s[j2], s[i2]
		k := s[(int(s[i2])+int(s[j2]))%256]
		result = append(result, dataBytes[n]^k)
	}

	if decrypt {
		if useGBK {
			return gbkToUTF8(result), nil
		}
		return string(result), nil
	}

	return hex.EncodeToString(result), nil
}

// utf8ToGBK 将UTF-8字符串转换为GBK字节
func utf8ToGBK(s string) []byte {
	var result []byte
	for _, r := range s {
		if r < 128 {
			result = append(result, byte(r))
		} else {
			result = append(result, gbkEncode(r)...)
		}
	}
	return result
}

// gbkToUTF8 将GBK字节转换为UTF-8字符串
func gbkToUTF8(data []byte) string {
	var result []rune
	i := 0
	for i < len(data) {
		if data[i] < 128 {
			result = append(result, rune(data[i]))
			i++
		} else if i+1 < len(data) {
			r := gbkDecode(data[i], data[i+1])
			result = append(result, r)
			i += 2
		} else {
			result = append(result, rune(data[i]))
			i++
		}
	}
	return string(result)
}

// gbkEncode 将Unicode字符编码为GBK字节
func gbkEncode(r rune) []byte {
	if r >= 0x4E00 && r <= 0x9FA5 {
		offset := r - 0x4E00
		high := byte(0xB0 + (offset / 94))
		low := byte(0xA1 + (offset % 94))
		return []byte{high, low}
	}
	return []byte{byte(r)}
}

// gbkDecode 将GBK字节解码为Unicode字符
func gbkDecode(high, low byte) rune {
	if high >= 0xB0 && high <= 0xF7 && low >= 0xA1 && low <= 0xFE {
		offset := int(high-0xB0)*94 + int(low-0xA1)
		return rune(0x4E00 + offset)
	}
	return rune(high)<<8 | rune(low)
}

// ParseRSAPrivateKey 解析RSA私钥PEM格式字符串
//
// 参数:
//   - privateKeyPEM: PEM格式的私钥字符串
//
// 返回:
//   - *rsa.PrivateKey: 解析后的RSA私钥
//   - error: 解析失败时的错误信息
func ParseRSAPrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing key")
	}

	priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	rsaPriv, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA private key")
	}

	return rsaPriv, nil
}

// ParseRSAPublicKey 解析RSA公钥PEM格式字符串
//
// 参数:
//   - publicKeyPEM: PEM格式的公钥字符串
//
// 返回:
//   - *rsa.PublicKey: 解析后的RSA公钥
//   - error: 解析失败时的错误信息
func ParseRSAPublicKey(publicKeyPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return rsaPub, nil
}

// RSAPrivateEncrypt 使用RSA私钥加密数据（与PHP openssl_private_encrypt兼容）
//
// 参数:
//   - data: 要加密的数据
//   - privateKeyPEM: PEM格式的私钥字符串
//
// 返回:
//   - string: 加密后的Base64字符串
//   - error: 加密失败时的错误信息
func RSAPrivateEncrypt(data string, privateKeyPEM string) (string, error) {
	privKey, err := ParseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}

	keySize := privKey.N.BitLen() / 8
	maxEncryptBlock := keySize - 11

	dataBytes := []byte(data)
	var encrypted []byte

	for i := 0; i < len(dataBytes); i += maxEncryptBlock {
		end := i + maxEncryptBlock
		if end > len(dataBytes) {
			end = len(dataBytes)
		}
		chunk := dataBytes[i:end]

		encryptedChunk, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.Hash(0), chunk)
		if err != nil {
			return "", fmt.Errorf("RSA private encrypt failed: %v", err)
		}
		encrypted = append(encrypted, encryptedChunk...)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// RSAPrivateDecrypt 使用RSA私钥解密数据（与PHP openssl_private_decrypt兼容）
//
// 参数:
//   - ciphertext: 要解密的Base64字符串
//   - privateKeyPEM: PEM格式的私钥字符串
//
// 返回:
//   - string: 解密后的原始数据
//   - error: 解密失败时的错误信息
func RSAPrivateDecrypt(ciphertext string, privateKeyPEM string) (string, error) {
	privKey, err := ParseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}

	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %v", err)
	}

	keySize := privKey.N.BitLen() / 8

	var decrypted []byte
	for i := 0; i < len(decoded); i += keySize {
		end := i + keySize
		if end > len(decoded) {
			end = len(decoded)
		}
		chunk := decoded[i:end]

		decryptedChunk, err := rsa.DecryptPKCS1v15(rand.Reader, privKey, chunk)
		if err != nil {
			return "", fmt.Errorf("RSA private decrypt failed: %v", err)
		}
		decrypted = append(decrypted, decryptedChunk...)
	}

	return string(decrypted), nil
}

// RSAPublicEncrypt 使用RSA公钥加密数据（OAEP填充）
//
// 参数:
//   - data: 要加密的数据
//   - publicKeyPEM: PEM格式的公钥字符串
//
// 返回:
//   - string: 加密后的Base64字符串
//   - error: 加密失败时的错误信息
func RSAPublicEncrypt(data string, publicKeyPEM string) (string, error) {
	pubKey, err := ParseRSAPublicKey(publicKeyPEM)
	if err != nil {
		return "", err
	}

	hash := sha1.New()
	rng := rand.Reader
	ciphertext, err := rsa.EncryptOAEP(hash, rng, pubKey, []byte(data), nil)
	if err != nil {
		return "", fmt.Errorf("RSA public encrypt failed: %v", err)
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// ==================== 客户端ID生成函数 ====================

// GenerateClientID 生成客户端唯一标识符
//
// 生成格式: CLIENT-XXXXXXXX-XXXXXXXX
// 该ID基于时间戳和随机数生成，每次调用生成不同的ID
//
// 返回:
//   - string: 生成的客户端唯一标识符
//
// 示例:
//
//	clientID := donghao.GenerateClientID()
//	fmt.Println("客户端ID:", clientID) // 输出: CLIENT-A3F9B2C1-D8E5A7B4
func GenerateClientID() string {
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)

	part1 := fmt.Sprintf("%08X", timestamp>>32)
	part2 := fmt.Sprintf("%08X", timestamp&0xFFFFFFFF)

	return fmt.Sprintf("CLIENT-%s-%s", part1, part2)
}

// GenerateDeviceID 生成设备唯一标识符
//
// 基于机器信息生成设备唯一标识符，格式: DEVICE-XXXXXXXX-XXXXXXXX-XXXXXXXX
// 该ID基于机器特征生成，同一台机器生成的ID相同
//
// 参数:
//   - macAddr: MAC地址（可选，用于增强唯一性）
//   - cpuInfo: CPU信息（可选，用于增强唯一性）
//   - diskInfo: 磁盘信息（可选，用于增强唯一性）
//
// 返回:
//   - string: 生成的设备唯一标识符
//
// 示例:
//
//	deviceID := donghao.GenerateDeviceID("00:11:22:33:44:55", "Intel-i7", "C:")
//	fmt.Println("设备ID:", deviceID) // 输出: DEVICE-A3F9B2C1-D8E5A7B4-12345678
func GenerateDeviceID(macAddr, cpuInfo, diskInfo string) string {
	machineInfo := fmt.Sprintf("%s|%s|%s|%d", macAddr, cpuInfo, diskInfo, time.Now().Year())

	hash := md5.Sum([]byte(machineInfo))
	hashStr := hex.EncodeToString(hash[:])

	return fmt.Sprintf("DEVICE-%s-%s-%s",
		hashStr[0:8],
		hashStr[8:16],
		hashStr[16:24])
}

// GenerateUUID 生成UUID格式的唯一标识符
//
// 生成符合RFC 4122标准的UUID v4格式标识符
// 格式: XXXXXXXX-XXXX-4XXX-YXXX-XXXXXXXXXXXX
//
// 返回:
//   - string: 生成的UUID
//
// 示例:
//
//	uuid := donghao.GenerateUUID()
//	fmt.Println("UUID:", uuid) // 输出: 550e8400-e29b-41d4-a716-446655440000
func GenerateUUID() string {
	u := make([]byte, 16)
	rand.Read(u)

	u[6] = (u[6] & 0x0f) | 0x40
	u[8] = (u[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

// GetClientIDFromStorage 从存储中获取或生成客户端ID
//
// 如果存储中已有客户端ID，则返回已有ID；
// 如果没有，则生成新的ID并返回
//
// 参数:
//   - storageKey: 存储键名（用于区分不同应用的客户端ID）
//
// 返回:
//   - string: 客户端ID
//   - bool: 是否为新生成的ID
//
// 注意: 此函数仅生成ID，实际存储需要调用方实现
//
// 示例:
//
//	clientID, isNew := donghao.GetClientIDFromStorage("myapp_client_id")
//	if isNew {
//	    fmt.Println("生成新的客户端ID:", clientID)
//	} else {
//	    fmt.Println("使用已有的客户端ID:", clientID)
//	}
func GetClientIDFromStorage(storageKey string) (string, bool) {
	newID := GenerateClientID()
	return newID, true
}

// ==================== 硬件设备标识生成函数 ====================

// GetMachineCode 获取机器唯一码（跨平台）
//
// 根据操作系统自动选择合适的方式生成设备唯一标识符：
//   - Windows: 基于CPU、主板、硬盘、MAC地址、BIOS等硬件信息
//   - Android: 基于Android ID、设备序列号、IMEI等信息
//   - 其他系统: 基于系统信息生成备用码
//
// 返回:
//   - string: 机器唯一码，格式 XXXX-XXXX-XXXX-XXXX
//   - error: 获取失败时的错误信息
//
// 示例:
//
//	code, err := donghao.GetMachineCode()
//	if err != nil {
//	    log.Printf("获取机器码失败: %v", err)
//	}
//	fmt.Println("机器码:", code) // 输出: A3F9-B2C1-D8E5-A7B4
func GetMachineCode() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return getWindowsMachineCode()
	case "android":
		return getAndroidMachineCode()
	default:
		return getFallbackMachineCode(), nil
	}
}

// GetHardwareID 获取硬件ID（跨平台）
//
// 生成更详细的硬件标识符
//
// 返回:
//   - string: 硬件ID，格式 HWID-XXXXXXXX-XXXXXXXX-XXXXXXXX-XXXXXXXX
//   - error: 获取失败时的错误信息
func GetHardwareID() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return getWindowsHardwareID()
	case "android":
		return getAndroidHardwareID()
	default:
		return getFallbackHardwareID(), nil
	}
}

// GetHardwareInfo 获取详细硬件信息（跨平台）
//
// 返回各种硬件的详细信息
//
// 返回:
//   - map[string]string: 硬件信息map
//   - error: 获取失败时的错误信息
func GetHardwareInfo() (map[string]string, error) {
	switch runtime.GOOS {
	case "windows":
		return getWindowsHardwareInfo()
	case "android":
		return getAndroidHardwareInfo()
	default:
		return getFallbackHardwareInfo(), nil
	}
}

// GetMachineCodeSafe 安全获取机器码（跨平台，带备用方案）
//
// 优先尝试获取硬件机器码，如果失败则使用备用方案
// 不会返回错误，始终返回一个标识符
//
// 返回:
//   - string: 机器码
func GetMachineCodeSafe() string {
	code, err := GetMachineCode()
	if err == nil && code != "" {
		return code
	}
	return getFallbackMachineCode()
}

// ==================== Windows 硬件信息获取 ====================

// getWindowsMachineCode 获取Windows机器码
func getWindowsMachineCode() (string, error) {
	cpuID, _ := getWMIValue("cpu", "ProcessorId")
	boardID, _ := getWMIValue("baseboard", "SerialNumber")
	diskID, _ := getWMIValue("diskdrive", "SerialNumber")
	macAddr, _ := getWindowsMACAddress()
	biosID, _ := getWMIValue("bios", "SerialNumber")

	hardwareInfo := fmt.Sprintf("%s|%s|%s|%s|%s", cpuID, boardID, diskID, macAddr, biosID)

	hash := md5.Sum([]byte(hardwareInfo))
	hashStr := hex.EncodeToString(hash[:])

	machineCode := fmt.Sprintf("%s-%s-%s-%s",
		hashStr[0:4],
		hashStr[4:8],
		hashStr[8:12],
		hashStr[12:16])

	finalHash := md5.Sum([]byte(machineCode))
	finalHashStr := hex.EncodeToString(finalHash[:])

	finalCode := fmt.Sprintf("%s-%s-%s-%s",
		finalHashStr[0:4],
		finalHashStr[4:8],
		finalHashStr[8:12],
		finalHashStr[12:16])

	return strings.ToUpper(finalCode), nil
}

// getWindowsHardwareID 获取Windows硬件ID
func getWindowsHardwareID() (string, error) {
	cpuID, _ := getWMIValue("cpu", "ProcessorId")
	boardID, _ := getWMIValue("baseboard", "SerialNumber")
	diskID, _ := getWMIValue("diskdrive", "SerialNumber")
	macAddr, _ := getWindowsMACAddress()
	biosID, _ := getWMIValue("bios", "SerialNumber")
	uuid, _ := getWMIValue("csproduct", "UUID")

	hardwareInfo := fmt.Sprintf("%s|%s|%s|%s|%s|%s", cpuID, boardID, diskID, macAddr, biosID, uuid)
	hash := md5.Sum([]byte(hardwareInfo))
	hashStr := hex.EncodeToString(hash[:])

	return fmt.Sprintf("HWID-%s-%s-%s-%s",
		hashStr[0:8],
		hashStr[8:16],
		hashStr[16:24],
		hashStr[24:32]), nil
}

// getWindowsHardwareInfo 获取Windows硬件信息
func getWindowsHardwareInfo() (map[string]string, error) {
	info := make(map[string]string)

	if cpu, err := getWMIValue("cpu", "ProcessorId"); err == nil && cpu != "" {
		info["cpu"] = cpu
	}
	if board, err := getWMIValue("baseboard", "SerialNumber"); err == nil && board != "" {
		info["board"] = board
	}
	if disk, err := getWMIValue("diskdrive", "SerialNumber"); err == nil && disk != "" {
		info["disk"] = disk
	}
	if mac, err := getWindowsMACAddress(); err == nil && mac != "" {
		info["mac"] = mac
	}
	if bios, err := getWMIValue("bios", "SerialNumber"); err == nil && bios != "" {
		info["bios"] = bios
	}
	if uuid, err := getWMIValue("csproduct", "UUID"); err == nil && uuid != "" {
		info["uuid"] = uuid
	}

	return info, nil
}

// getWMIValue 通过WMI获取硬件信息
func getWMIValue(class, property string) (string, error) {
	cmd := exec.Command("wmic", class, "get", property, "/value")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	result := parseWMIOutput(string(output), property)
	return cleanHardwareString(result), nil
}

// getWindowsMACAddress 获取Windows MAC地址
func getWindowsMACAddress() (string, error) {
	cmd := exec.Command("wmic", "nic", "where", "NetConnectionStatus=2", "get", "MACAddress", "/value")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return parseWMIOutput(string(output), "MACAddress"), nil
}

// parseWMIOutput 解析WMI输出
func parseWMIOutput(output, key string) string {
	pattern := fmt.Sprintf(`(?i)%s\s*=\s*(\S+)`, regexp.QuoteMeta(key))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(output)

	if len(matches) > 1 {
		return matches[1]
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), strings.ToUpper(key)+"=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}

// ==================== Android 硬件信息获取 ====================

// getAndroidMachineCode 获取Android机器码
func getAndroidMachineCode() (string, error) {
	androidID, _ := getAndroidProperty("ro.serialno")
	if androidID == "" {
		androidID, _ = getAndroidProperty("ro.boot.serialno")
	}

	deviceID, _ := getAndroidProperty("ro.product.device")
	brand, _ := getAndroidProperty("ro.product.brand")
	model, _ := getAndroidProperty("ro.product.model")
	board, _ := getAndroidProperty("ro.product.board")

	deviceInfo := fmt.Sprintf("%s|%s|%s|%s|%s", androidID, deviceID, brand, model, board)

	hash := md5.Sum([]byte(deviceInfo))
	hashStr := hex.EncodeToString(hash[:])

	machineCode := fmt.Sprintf("%s-%s-%s-%s",
		hashStr[0:4],
		hashStr[4:8],
		hashStr[8:12],
		hashStr[12:16])

	finalHash := md5.Sum([]byte(machineCode))
	finalHashStr := hex.EncodeToString(finalHash[:])

	finalCode := fmt.Sprintf("%s-%s-%s-%s",
		finalHashStr[0:4],
		finalHashStr[4:8],
		finalHashStr[8:12],
		finalHashStr[12:16])

	return strings.ToUpper(finalCode), nil
}

// getAndroidHardwareID 获取Android硬件ID
func getAndroidHardwareID() (string, error) {
	androidID, _ := getAndroidProperty("ro.serialno")
	if androidID == "" {
		androidID, _ = getAndroidProperty("ro.boot.serialno")
	}

	deviceID, _ := getAndroidProperty("ro.product.device")
	brand, _ := getAndroidProperty("ro.product.brand")
	model, _ := getAndroidProperty("ro.product.model")
	board, _ := getAndroidProperty("ro.product.board")
	manufacturer, _ := getAndroidProperty("ro.product.manufacturer")
	hardware, _ := getAndroidProperty("ro.hardware")

	deviceInfo := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s", androidID, deviceID, brand, model, board, manufacturer, hardware)
	hash := md5.Sum([]byte(deviceInfo))
	hashStr := hex.EncodeToString(hash[:])

	return fmt.Sprintf("HWID-%s-%s-%s-%s",
		hashStr[0:8],
		hashStr[8:16],
		hashStr[16:24],
		hashStr[24:32]), nil
}

// getAndroidHardwareInfo 获取Android硬件信息
func getAndroidHardwareInfo() (map[string]string, error) {
	info := make(map[string]string)

	props := map[string]string{
		"serialno":     "ro.serialno",
		"device":       "ro.product.device",
		"brand":        "ro.product.brand",
		"model":        "ro.product.model",
		"board":        "ro.product.board",
		"manufacturer": "ro.product.manufacturer",
		"hardware":     "ro.hardware",
		"android_id":   "ro.boot.android_id",
	}

	for key, prop := range props {
		if value, err := getAndroidProperty(prop); err == nil && value != "" {
			info[key] = value
		}
	}

	return info, nil
}

// getAndroidProperty 获取Android系统属性
func getAndroidProperty(prop string) (string, error) {
	cmd := exec.Command("getprop", prop)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ==================== 备用方案 ====================

// getFallbackMachineCode 获取备用机器码
func getFallbackMachineCode() string {
	hostname, _ := os.Hostname()
	fallback := fmt.Sprintf("%s|%s|%s|%d", runtime.GOOS, runtime.GOARCH, hostname, time.Now().Year())

	hash := md5.Sum([]byte(fallback))
	hashStr := hex.EncodeToString(hash[:])

	machineCode := fmt.Sprintf("%s-%s-%s-%s",
		hashStr[0:4],
		hashStr[4:8],
		hashStr[8:12],
		hashStr[12:16])

	finalHash := md5.Sum([]byte(machineCode))
	finalHashStr := hex.EncodeToString(finalHash[:])

	return fmt.Sprintf("%s-%s-%s-%s",
		finalHashStr[0:4],
		finalHashStr[4:8],
		finalHashStr[8:12],
		finalHashStr[12:16])
}

// getFallbackHardwareID 获取备用硬件ID
func getFallbackHardwareID() string {
	hostname, _ := os.Hostname()
	fallback := fmt.Sprintf("%s|%s|%s|%d|%d", runtime.GOOS, runtime.GOARCH, hostname, time.Now().Year(), time.Now().Unix())
	hash := md5.Sum([]byte(fallback))
	hashStr := hex.EncodeToString(hash[:])

	return fmt.Sprintf("HWID-%s-%s-%s-%s",
		hashStr[0:8],
		hashStr[8:16],
		hashStr[16:24],
		hashStr[24:32])
}

// getFallbackHardwareInfo 获取备用硬件信息
func getFallbackHardwareInfo() map[string]string {
	info := make(map[string]string)
	info["os"] = runtime.GOOS
	info["arch"] = runtime.GOARCH
	info["hostname"], _ = os.Hostname()
	info["cpu_num"] = fmt.Sprintf("%d", runtime.NumCPU())
	return info
}

// cleanHardwareString 清理硬件信息字符串
func cleanHardwareString(s string) string {
	s = strings.TrimSpace(s)
	invalidValues := []string{
		"To be filled by O.E.M.",
		"To Be Filled By O.E.M.",
		"Not Available",
		"None",
		"Default string",
		"System Serial Number",
		"Base Board Serial Number",
		"unknown",
	}

	for _, invalid := range invalidValues {
		if strings.EqualFold(s, invalid) {
			return ""
		}
	}

	return s
}

// ==================== AES-GCM 加密函数 ====================

// padKeyTo32Bytes 将密钥填充或截断为32字节
func padKeyTo32Bytes(key string) []byte {
	keyBytes := []byte(key)
	if len(keyBytes) >= 32 {
		return keyBytes[:32]
	}
	padded := make([]byte, 32)
	copy(padded, keyBytes)
	return padded
}

// aesGcmEncrypt AES-256-GCM加密
//
// 参数:
//   - plaintext: 待加密数据
//   - key: 32字节密钥
//
// 返回:
//   - string: Base64编码的加密数据（Nonce + 密文 + Tag）
//   - error: 加密错误
func aesGcmEncrypt(plaintext string, key string) (string, error) {
	keyBytes := padKeyTo32Bytes(key)
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("AES cipher creation failed: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("GCM mode creation failed: %v", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", fmt.Errorf("nonce generation failed: %v", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// aesGcmDecrypt AES-256-GCM解密
//
// 参数:
//   - encoded: Base64编码的加密数据
//   - key: 32字节密钥
//
// 返回:
//   - string: 解密后的明文
//   - error: 解密错误
func aesGcmDecrypt(encoded string, key string) (string, error) {
	keyBytes := padKeyTo32Bytes(key)
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %v", err)
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("AES cipher creation failed: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("GCM mode creation failed: %v", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %v", err)
	}
	return string(plaintext), nil
}

// ==================== ChaCha20-Poly1305 加密函数 - 已禁用 ====================
// OpenSSL 1.1.1 不支持 AEAD Tag，导致加密解密不兼容

// chachaEncrypt ChaCha20-Poly1305加密
//
// 参数:
//   - plaintext: 待加密数据
//   - key: 32字节密钥
//
// 返回:
//   - string: Base64编码的加密数据（Nonce + 密文 + Tag）
//   - error: 加密错误
// func chachaEncrypt(plaintext string, key string) (string, error) {
// 	keyBytes := padKeyTo32Bytes(key)
// 	aead, err := chacha20poly1305.NewX(keyBytes)
// 	if err != nil {
// 		return "", fmt.Errorf("ChaCha20-Poly1305 creation failed: %v", err)
// 	}
// 	nonce := make([]byte, aead.NonceSize())
// 	if _, err = rand.Read(nonce); err != nil {
// 		return "", fmt.Errorf("nonce generation failed: %v", err)
// 	}
// 	ciphertext := aead.Seal(nonce, nonce, []byte(plaintext), nil)
// 	return base64.StdEncoding.EncodeToString(ciphertext), nil
// }
// 
// chachaDecrypt ChaCha20-Poly1305解密
//
// 参数:
//   - encoded: Base64编码的加密数据
//   - key: 32字节密钥
//
// 返回:
//   - string: 解密后的明文
//   - error: 解密错误
// func chachaDecrypt(encoded string, key string) (string, error) {
// 	keyBytes := padKeyTo32Bytes(key)
// 	data, err := base64.StdEncoding.DecodeString(encoded)
// 	if err != nil {
// 		return "", fmt.Errorf("base64 decode failed: %v", err)
// 	}
// 	aead, err := chacha20poly1305.NewX(keyBytes)
// 	if err != nil {
// 		return "", fmt.Errorf("ChaCha20-Poly1305 creation failed: %v", err)
// 	}
// 	nonceSize := aead.NonceSize()
// 	if len(data) < nonceSize {
// 		return "", errors.New("ciphertext too short")
// 	}
// 	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
// 	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
// 	if err != nil {
// 		return "", fmt.Errorf("decryption failed: %v", err)
// 	}
// 	return string(plaintext), nil
// }
