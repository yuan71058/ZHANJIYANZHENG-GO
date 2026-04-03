# 冬浩验证系统 - Go SDK API 文档

**版本**: 1.0  
**更新日期**: 2026-04-04  
**Gitee**: https://gitee.com/yuan71058/dong-hao-verification

---

## 目录

- [快速开始](#快速开�?
- [Client 客户端](#client-客户�?
- [Result 结果对象](#result-结果对象)
- [用户管理接口](#用户管理接口)
- [卡密管理接口](#卡密管理接口)
- [数据管理接口](#数据管理接口)
- [云变量接口](#云变量接�?
- [云计算接口](#云计算接�?
- [系统接口](#系统接口)
- [设备标识生成](#设备标识生成)
- [加密工具函数](#加密工具函数)
- [常量定义](#常量定义)

---

## 快速开�?
### 安装

```bash
go get gitee.com/yuan71058/dong-hao-verification
```

### 基本使用

```go
package main

import (
    "fmt"
    "log"
    "gitee.com/yuan71058/dong-hao-verification"
)

func main() {
    // 创建客户�?    client := donghao.NewClient("http://your-domain.com", 1)
    client.SetTimeout(30)
    
    // 配置加密（可选）
    client.SetEncryption(donghao.ENC_RC4, "your-rc4-key")
    
    // 配置签名（可选）
    client.SetSignConfig("your-app-key", "[data]123[key]456", true)
    
    // 用户登录
    result, err := client.Login("username", "password", "1.0", "mac", "ip", "clientid")
    if err != nil {
        log.Fatal(err)
    }
    
    if result.IsSuccess() {
        fmt.Println("登录成功，Token:", client.GetToken())
    } else {
        fmt.Println("登录失败:", result.Msg())
    }
}
```

---

## Client 客户�?
### NewClient

创建一个新的API客户�?
```go
func NewClient(baseURL string, appID int) *Client
```

**参数**:
- `baseURL`: API服务器基础URL，如 `"http://your-domain.com"`
- `appID`: 软件ID，从管理后台获取

**返回**:
- `*Client`: 客户端实�?
**示例**:
```go
client := donghao.NewClient("http://your-domain.com", 1)
```

---

### SetTimeout

设置HTTP请求超时时间

```go
func (c *Client) SetTimeout(seconds int)
```

**参数**:
- `seconds`: 超时时间（秒�?
**示例**:
```go
client.SetTimeout(30) // 30秒超�?```

---

### SetEncryption

设置加密配置

```go
func (c *Client) SetEncryption(encType int, key string)
```

**参数**:
- `encType`: 加密类型 (`ENC_NONE`, `ENC_RC4`, `ENC_RSA`, `ENC_BASE64`, `ENC_CUSTOM`)
- `key`: 加密密钥

**示例**:
```go
client.SetEncryption(donghao.ENC_RC4, "qHBqZ-vsGzS-32oiT-trIpg-iJhR")
```

---

### SetSignConfig

设置签名配置

```go
func (c *Client) SetSignConfig(appKey, template string, needSign bool)
```

**参数**:
- `appKey`: 应用密钥
- `template`: 签名模板，如 `"[data]123[key]456"`
- `needSign`: 是否需要签�?
**示例**:
```go
client.SetSignConfig("hbjBpRbXqNnMwbS3", "[data]123[key]456", true)
```

---

### SetHeartbeatInterval

设置自动心跳间隔

```go
func (c *Client) SetHeartbeatInterval(seconds int)
```

**参数**:
- `seconds`: 心跳间隔（秒），默认60�?
**示例**:
```go
client.SetHeartbeatInterval(60)
```

---

### GetToken

获取当前登录Token

```go
func (c *Client) GetToken() string
```

**返回**:
- `string`: 当前Token，未登录时返回空字符�?
---

### GetCurrentUser

获取当前登录用户�?
```go
func (c *Client) GetCurrentUser() string
```

**返回**:
- `string`: 当前用户名，未登录时返回空字符串

---

## Result 结果对象

### IsSuccess

检查操作是否成�?
```go
func (r *Result) IsSuccess() bool
```

**返回**:
- `bool`: true表示成功，false表示失败

---

### Msg

获取错误消息

```go
func (r *Result) Msg() string
```

**返回**:
- `string`: 错误消息，成功时返回空字符串

---

### GetResultMap

获取结果数据为map

```go
func (r *Result) GetResultMap() map[string]interface{}
```

**返回**:
- `map[string]interface{}`: 结果数据map

---

### GetData

获取解密后的数据字段

```go
func (r *Result) GetData() (string, error)
```

**返回**:
- `string`: 解密后的数据
- `error`: 解密错误

---

### GetGroupData

获取解密后的用户组数�?
```go
func (r *Result) GetGroupData() (string, error)
```

**返回**:
- `string`: 解密后的用户组数�?- `error`: 解密错误

---

### GetVariableValue

获取云变量�?
```go
func (r *Result) GetVariableValue() (string, error)
```

**返回**:
- `string`: 云变量�?- `error`: 获取错误

---

### GetTokenID

获取Token ID

```go
func (r *Result) GetTokenID() string
```

**返回**:
- `string`: Token ID

---

## 用户管理接口

### Login

用户登录

```go
func (c *Client) Login(user, pwd, ver, mac, ip, clientid string) (*Result, error)
```

**参数**:
- `user`: 用户�?- `pwd`: 密码
- `ver`: 软件版本�?- `mac`: 机器�?MAC地址
- `ip`: 客户端IP地址
- `clientid`: 客户端ID

**返回**:
- `*Result`: 登录结果
- `error`: 请求错误

**示例**:
```go
result, err := client.Login("testuser", "password", "1.0", "00:11:22:33:44:55", "192.168.1.1", "client001")
if result.IsSuccess() {
    fmt.Println("登录成功，Token:", client.GetToken())
}
```

---

### LoginCard

卡密登录（无需用户名密码）

```go
func (c *Client) LoginCard(card, ver, mac, ip, clientid string) (*Result, error)
```

**参数**:
- `card`: 卡密
- `ver`: 软件版本�?- `mac`: 机器�?MAC地址
- `ip`: 客户端IP地址
- `clientid`: 客户端ID

**返回**:
- `*Result`: 登录结果
- `error`: 请求错误

**示例**:
```go
result, err := client.LoginCard("YKUvGvYSuVFFTp41T1WO", "1.0", "mac", "ip", "client001")
if result.IsSuccess() {
    fmt.Println("卡密登录成功，Token:", client.GetToken())
}
```

---

### Reg

用户注册

```go
func (c *Client) Reg(user, pwd, card, userqq, email, tjr, ver, mac, ip, clientid string) (*Result, error)
```

**参数**:
- `user`: 用户�?- `pwd`: 密码
- `card`: 卡密（可选）
- `userqq`: QQ号（可选）
- `email`: 邮箱（可选）
- `tjr`: 推荐人（可选）
- `ver`: 软件版本�?- `mac`: 机器�?- `ip`: IP地址
- `clientid`: 客户端ID

**示例**:
```go
result, err := client.Reg("newuser", "password", "CARD-123", "123456", "user@example.com", "", "1.0", "mac", "ip", "client001")
```

---

### Logout

用户退出登�?
```go
func (c *Client) Logout(user, tokenid, ver, mac, ip, clientid string) (*Result, error)
```

**示例**:
```go
result, err := client.Logout("testuser", client.GetToken(), "1.0", "mac", "ip", "client001")
```

---

### GetUser

获取用户信息

```go
func (c *Client) GetUser(user, tokenid, ver, mac, ip, clientid string) (*Result, error)
```

**示例**:
```go
result, err := client.GetUser("testuser", client.GetToken(), "1.0", "mac", "ip", "client001")
if result.IsSuccess() {
    m := result.GetResultMap()
    if endtime, ok := m["endtime"].(string); ok {
        fmt.Println("到期时间:", endtime)
    }
}
```

---

### Uppwd

修改密码

```go
func (c *Client) Uppwd(user, pwd, newpwd, ver, mac, ip, clientid string) (*Result, error)
```

**示例**:
```go
result, err := client.Uppwd("testuser", "oldpwd", "newpwd", "1.0", "mac", "ip", "client001")
```

---

### Heartbeat

发送心�?
```go
func (c *Client) Heartbeat(user, tokenid, ver, mac, ip, clientid string) (*Result, error)
```

**示例**:
```go
result, err := client.Heartbeat("testuser", client.GetToken(), "1.0", "mac", "ip", "client001")
```

---

### StartAutoHeartbeat（已禁用�?
> ⚠️ **注意**: 自动心跳功能存在问题，已禁用。请使用手动心跳 `Heartbeat()` 方法代替�?
启动自动心跳

```go
// func (c *Client) StartAutoHeartbeat(user, tokenid, ver, mac, ip, clientid string)
```

**手动心跳示例**:
```go
result, err := client.Heartbeat("testuser", client.GetToken(), "1.0", "mac", "ip", "client001")
```

---

### StopAutoHeartbeat（已禁用�?
停止自动心跳

```go
// func (c *Client) StopAutoHeartbeat()
```

---

## 卡密管理接口

### Recharge

卡密充�?
```go
func (c *Client) Recharge(user, card, ver, mac, ip, clientid string) (*Result, error)
```

**参数**:
- `user`: 用户�?- `card`: 卡密
- `ver`: 软件版本�?- `mac`: 机器�?- `ip`: IP地址
- `clientid`: 客户端ID

**示例**:
```go
result, err := client.Recharge("testuser", "YKUvGvYSuVFFTp41T1WO", "1.0", "mac", "ip", "client001")
if result.IsSuccess() {
    fmt.Println("充值成�?)
}
```

---

### Binding

绑定换绑（修改用户信息）

```go
func (c *Client) Binding(user, pwd, newuser, newmac, newip, newuserqq, ver, mac, ip, clientid string) (*Result, error)
```

**参数**:
- `user`: 用户名
- `pwd`: 密码
- `newuser`: 新用户名（可选，为空则不修改）
- `newmac`: 新机器码（可选，为空则不修改）
- `newip`: 新IP地址（可选，为空则不修改）
- `newuserqq`: 新QQ号（可选，为空则不修改）
- `ver`: 软件版本号
- `mac`: 机器码
- `ip`: 客户端IP地址
- `clientid`: 客户端ID

**示例**:
```go
result, err := client.Binding("testuser", "password", "newuser", "newmac", "newip", "newqq", "1.0", "mac", "ip", "client001")
```

---

## 数据管理接口

### GetUdata

获取用户数据

```go
func (c *Client) GetUdata(user, tokenid, ver, mac, ip, clientid string) (*Result, error)
```

**示例**:
```go
result, err := client.GetUdata("testuser", client.GetToken(), "1.0", "mac", "ip", "client001")
if result.IsSuccess() {
    data, _ := result.GetData()
    fmt.Println("用户数据:", data)
}
```

---

### SetUdata

设置用户数据

```go
func (c *Client) SetUdata(user, tokenid, udata, ver, mac, ip, clientid string) (*Result, error)
```

**示例**:
```go
jsonData := `{"level": 5, "exp": 1000}`
result, err := client.SetUdata("testuser", client.GetToken(), jsonData, "1.0", "mac", "ip", "client001")
```

---

### GetUdata2

获取用户数据2（第二数据区�?
```go
func (c *Client) GetUdata2(user, tokenid, ver, mac, ip, clientid string) (*Result, error)
```

---

### SetUdata2

设置用户数据2（第二数据区�?
```go
func (c *Client) SetUdata2(user, tokenid, udata, ver, mac, ip, clientid string) (*Result, error)
```

---

## 云变量接�?
### GetVariable

获取云变�?
```go
func (c *Client) GetVariable(user, tokenid, cloudkey, ver, mac, ip, clientid string) (*Result, error)
```

**示例**:
```go
result, err := client.GetVariable("testuser", client.GetToken(), "config_key", "1.0", "mac", "ip", "client001")
if result.IsSuccess() {
    value, _ := result.GetVariableValue()
    fmt.Println("变量�?", value)
}
```

---

### SetVariable

设置云变�?
```go
func (c *Client) SetVariable(user, tokenid, cloudkey, cloudvalue, ver, mac, ip, clientid string) (*Result, error)
```

**示例**:
```go
result, err := client.SetVariable("testuser", client.GetToken(), "config_key", `{"enabled":true}`, "1.0", "mac", "ip", "client001")
```

---

### DelVariable

删除云变�?
```go
func (c *Client) DelVariable(user, tokenid, cloudkey, ver, mac, ip, clientid string) (*Result, error)
```

---

### Constant

获取云端常量

```go
func (c *Client) Constant(user, tokenid, cloudkey, ver, mac, ip, clientid string) (*Result, error)
```

---

## 云计算接�?
### Func

调用云计算函数（需登录�?
```go
func (c *Client) Func(user, tokenid, fun, para, ver, mac, ip, clientid string) (*Result, error)
```

**示例**:
```go
result, err := client.Func("testuser", client.GetToken(), "jia", "1,2", "1.0", "mac", "ip", "client001")
if result.IsSuccess() {
    fmt.Println("计算结果:", result.Data)
}
```

---

### Func2

调用云计算函�?（无需登录�?
```go
func (c *Client) Func2(fun, para, ver, mac, ip, clientid string) (*Result, error)
```

**示例**:
```go
result, err := client.Func2("jia", "1,2", "1.0", "mac", "ip", "client001")
```

---

### CallPHP

调用PHP函数（需登录�?
```go
func (c *Client) CallPHP(user, tokenid, fun, para, ver, mac, ip, clientid string) (*Result, error)
```

---

### CallPHP2

调用PHP函数2（无需登录�?
```go
func (c *Client) CallPHP2(fun, para, ver, mac, ip, clientid string) (*Result, error)
```

---

## 系统接口

### Init

软件初始�?
```go
func (c *Client) Init(ver, mac, ip, clientid string) (*Result, error)
```

**示例**:
```go
result, err := client.Init("1.0", "mac", "ip", "client001")
if result.IsSuccess() {
    m := result.GetResultMap()
    if name, ok := m["name"].(string); ok {
        fmt.Println("软件名称:", name)
    }
}
```

---

### Notice

获取公告

```go
func (c *Client) Notice(title, ver, mac, ip, clientid string) (*Result, error)
```

---

### Ver

获取版本信息

```go
func (c *Client) Ver(ver, mac, ip, clientid string) (*Result, error)
```

---

### GetBlack

查询黑名�?
```go
func (c *Client) GetBlack(bType, bData string) (*Result, error)
```

**参数**:
- `bType`: 查询类型 (ip/mac/user)
- `bData`: 查询数据

**示例**:
```go
result, err := client.GetBlack("ip", "192.168.1.1")
```

---

### SetBlack

设置黑名�?
```go
func (c *Client) SetBlack(bType, bData, bBz, ver, mac, ip, clientid string) (*Result, error)
```

---

### AddLog

添加日志

```go
func (c *Client) AddLog(user, infos, ver, mac, ip, clientid string) (*Result, error)
```

---

### DeductPoints

扣除点数

```go
func (c *Client) DeductPoints(user, tokenid string, sl int, ver, mac, ip, clientid string) (*Result, error)
```

---

### Bindreferrer

绑定推荐�?
```go
func (c *Client) Bindreferrer(user, pwd, tjr, ver, mac, ip, clientid string) (*Result, error)
```

---

### Relay

中转请求

```go
func (c *Client) Relay(params map[string]string, ver, mac, ip, clientid string) (*Result, error)
```

---

## 设备标识生成

### GetMachineCode

获取机器唯一码（跨平台）�?**推荐**

```go
func GetMachineCode() (string, error)
```

**说明**:
- **Windows**: 基于CPU序列号、主板序列号、硬盘序列号、MAC地址、BIOS信息
- **Android**: 基于设备序列号、设备型号、品牌、主板等信息
- **其他系统**: 基于系统信息生成备用�?- 同一台设备每次生成的码相�?- 已集成到SDK中，无需额外文件
- **采用双层MD5加密，增强安全�?*

**加密流程**:
```
硬件信息 �?MD5(第一�? �?中间�?�?MD5(第二�? �?最终机器码
```

**返回**:
- `string`: 机器唯一码，格式 `XXXX-XXXX-XXXX-XXXX`�?6位十六进制）
- `error`: 获取失败时的错误信息

**示例**:
```go
code, err := donghao.GetMachineCode()
if err != nil {
    log.Fatal("获取机器码失�?", err)
}
fmt.Println("机器�?", code) // 输出: A3F9-B2C1-D8E5-A7B4
```

---

### GetHardwareID

获取硬件ID（跨平台�?
```go
func GetHardwareID() (string, error)
```

**说明**:
- **Windows**: 包含CPU、主板、硬盘、MAC地址、BIOS、系统UUID
- **Android**: 包含设备序列号、型号、品牌、主板、制造商、硬件信�?- **其他系统**: 基于系统信息生成
- 已集成到SDK�?
**返回**:
- `string`: 硬件ID，格�?`HWID-XXXXXXXX-XXXXXXXX-XXXXXXXX-XXXXXXXX`
- `error`: 获取失败时的错误信息

**示例**:
```go
hwid, err := donghao.GetHardwareID()
if err != nil {
    log.Fatal(err)
}
fmt.Println("硬件ID:", hwid)
```

---

### GetHardwareInfo

获取详细硬件信息（跨平台�?
```go
func GetHardwareInfo() (map[string]string, error)
```

**说明**:
- **Windows**: 返回 cpu, board, disk, mac, bios, uuid
- **Android**: 返回 serialno, device, brand, model, board, manufacturer, hardware, android_id
- **其他系统**: 返回 os, arch, hostname, cpu_num
- 已集成到SDK�?
**返回**:
- `map[string]string`: 硬件信息map
- `error`: 获取失败时的错误信息

**示例**:
```go
info, err := donghao.GetHardwareInfo()
if err != nil {
    log.Fatal(err)
}
for key, value := range info {
    fmt.Printf("%s: %s\n", key, value)
}
```

---

### GetMachineCodeSafe

安全获取机器码（跨平台，带备用方案）

```go
func GetMachineCodeSafe() string
```

**说明**:
- 优先尝试获取硬件机器码（Windows/Android�?- 如果失败，则基于系统信息生成备用�?- 不会返回错误，始终返回一个标识符
- 已集成到SDK�?
**返回**:
- `string`: 机器�?
**示例**:
```go
code := donghao.GetMachineCodeSafe()
fmt.Println("机器�?", code)
```

---

### GenerateClientID

生成客户端唯一标识符（软件生成�?
```go
func GenerateClientID() string
```

**说明**:
- 基于时间戳和随机数生�?- 每次调用生成的ID不同
- 适用于临时标�?
**返回**:
- `string`: 客户端ID，格�?`CLIENT-XXXXXXXX-XXXXXXXX`

**示例**:
```go
clientID := donghao.GenerateClientID()
fmt.Println("客户端ID:", clientID) // CLIENT-A3F9B2C1-D8E5A7B4
```

---

### GenerateDeviceID

生成设备唯一标识符（基于硬件，跨平台�?
```go
func GenerateDeviceID() (string, error)
```

**说明**:
- 基于真实硬件信息生成设备唯一标识�?- **Windows**: 使用CPU、主板、硬盘等硬件信息
- **Android**: 使用设备序列号、型号等信息
- 同一台设备每次生成的ID相同
- 已集成到SDK�?
**返回**:
- `string`: 设备ID，格�?`XXXX-XXXX-XXXX-XXXX`
- `error`: 获取失败时的错误信息

**示例**:
```go
deviceID, err := donghao.GenerateDeviceID()
if err != nil {
    log.Fatal(err)
}
fmt.Println("设备ID:", deviceID)
```

---

### GenerateUUID

生成UUID

```go
func GenerateUUID() string
```

**返回**:
- `string`: UUID，格�?`XXXXXXXX-XXXX-4XXX-YXXX-XXXXXXXXXXXX`

**示例**:
```go
uuid := donghao.GenerateUUID()
fmt.Println("UUID:", uuid)
```

---

### GenerateMachineCode

生成机器�?
```go
func GenerateMachineCode(seed string) string
```

**参数**:
- `seed`: 种子字符串（如MAC地址�?
**返回**:
- `string`: 机器码，格式 `XXXX-XXXX-XXXX-XXXX`

**示例**:
```go
code := donghao.GenerateMachineCode("00:11:22:33:44:55")
fmt.Println("机器�?", code)
```

---

### GetClientIDFromStorage

从存储中获取或生成客户端ID

```go
func GetClientIDFromStorage(storageKey string) (string, bool)
```

**参数**:
- `storageKey`: 存储键名

**返回**:
- `string`: 客户端ID
- `bool`: 是否为新生成的ID

**注意**: 此函数仅生成ID，实际存储需要调用方实现

---

## 加密工具函数

### Encrypt

加密数据

```go
func (c *Client) Encrypt(data string) (string, error)
```

---

### Decrypt

解密数据

```go
func (c *Client) Decrypt(data string) (string, error)
```

---

### GenerateMD5

生成MD5哈希

```go
func GenerateMD5(text string) string
```

---

### GenerateSign

生成签名（不排序，保持参数原始顺序）

```go
func GenerateSign(params map[string]string, appKey string, template string) string
```

**签名算法说明**:
1. 获取所有请求参数（排除 `sign`、`act`、`appid`）
2. **保持参数原始顺序**（不进行排序）
3. 拼接为 `key1=value1&key2=value2&...`
4. 如果有签名模板：将 `[data]` 替换为拼接字符串，`[key]` 替换为 appKey
5. 如果无模板：直接在末尾追加 appKey
6. 对最终字符串进行 MD5 加密

**示例**:
```go
params := map[string]string{
    "user": "test",
    "pwd": "123456",
}
sign := donghao.GenerateSign(params, "your-key", "[data]123[key]")
// sign = MD5("user=test&pwd=123456123your-key")
```

---

### RC4Crypt

RC4加密/解密

```go
func RC4Crypt(data string, key string, decrypt bool) (string, error)
```

---

### RSAPrivateEncrypt

RSA私钥加密

```go
func RSAPrivateEncrypt(data string, privateKeyPEM string) (string, error)
```

---

### RSAPublicEncrypt

RSA公钥加密

```go
func RSAPublicEncrypt(data string, publicKeyPEM string) (string, error)
```

---

## 常量定义

### 加密类型

```go
const (
    ENC_NONE   = 0 // 不加�?    ENC_RC4    = 1 // RC4加密
    ENC_RSA    = 2 // RSA加密
    ENC_BASE64 = 3 // Base64编码
    ENC_CUSTOM = 4 // 自定义加�?)
```

---

## 错误处理

所有API调用都返�?`(*Result, error)`�?
- `error` 不为nil：表示HTTP请求失败（网络错误等�?- `error` 为nil�?`result.IsSuccess()` 为false：表示API返回错误（如密码错误、卡密无效等�?
**建议的错误处理方�?*:

```go
result, err := client.Login("user", "pwd", "1.0", "mac", "ip", "clientid")
if err != nil {
    // HTTP请求失败
    log.Printf("请求失败: %v", err)
    return
}

if !result.IsSuccess() {
    // API返回错误
    log.Printf("登录失败: %s", result.Msg())
    return
}

// 登录成功
fmt.Println("登录成功，Token:", client.GetToken())
```

---

## 完整示例

```go
package main

import (
    "fmt"
    "log"
    "gitee.com/yuan71058/dong-hao-verification"
)

func main() {
    // 创建客户�?    client := donghao.NewClient("http://your-domain.com", 1)
    client.SetTimeout(30)
    
    // 配置RC4加密
    client.SetEncryption(donghao.ENC_RC4, "qHBqZ-vsGzS-32oiT-trIpg-iJhR")
    
    // 配置签名
    client.SetSignConfig("hbjBpRbXqNnMwbS3", "[data]123[key]456", true)
    
    // 生成客户端ID
    clientID := donghao.GenerateClientID()
    fmt.Println("客户端ID:", clientID)
    
    // 软件初始�?    result, err := client.Init("1.0.0", "00:11:22:33:44:55", "192.168.1.100", clientID)
    if err != nil || !result.IsSuccess() {
        log.Fatal("初始化失�?)
    }
    
    // 卡密登录
    result, err = client.LoginCard("YKUvGvYSuVFFTp41T1WO", "1.0.0", "00:11:22:33:44:55", "192.168.1.100", clientID)
    if err != nil {
        log.Fatal("请求失败:", err)
    }
    
    if !result.IsSuccess() {
        log.Fatal("登录失败:", result.Msg())
    }
    
    fmt.Println("登录成功，Token:", client.GetToken())
    
    // 手动心跳（自动心跳已禁用�?    heartResult, _ := client.Heartbeat(client.GetCurrentUser(), client.GetToken(), "1.0.0", "00:11:22:33:44:55", "192.168.1.100", clientID)
    fmt.Println("心跳结果:", heartResult.IsSuccess())
    
    // 保持程序运行
    select {}
}
```

---

**文档版本**: 1.0  
**最后更新**: 2026-04-04
