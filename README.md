<div align="center">

# 冬浩验证系统 - Go SDK

![Version](https://img.shields.io/badge/version-1.4-blue.svg)
![Go](https://img.shields.io/badge/Go-1.25+-00ADD8.svg?logo=go)
![License](https://img.shields.io/badge/license-MIT-green.svg)

**冬浩验证系统 API 的完整 Go 语言封装**

[安装使用](#安装) | [快速开始](#快速开始) | [API 文档](#api-接口列表) | [示例代码](example/main.go)

</div>

---

## 更新日志

### v1.4 (2026-04-11)

#### Bug 修复
- **修复 httpPost token 参数被无条件覆盖的问题**
  - 问题原因：`httpPost()` 在注入 `currentToken` 时，会覆盖 `Heartbeat()` 等方法手动传入的 `token` 参数
  - 影响范围：所有通过 `httpPost` 发送请求且需要自定义 `token` 的接口
  - 修复方案：添加 `params["token"] == ""` 判断，只在参数中没有 token 时才自动注入
  ```go
  // 修复前
  if c.currentToken != "" {
      params["token"] = c.currentToken
  }
  
  // 修复后
  if c.currentToken != "" && params["token"] == "" {
      params["token"] = c.currentToken
  }
  ```

- **修复 heartbeatParams 字段命名误导**
  - 字段名 `TokenID` 改为 `Token`，与实际用途保持一致
  - 同步更新所有引用处（心跳循环、参数赋值）

#### 变更文件
| 文件 | 变更内容 |
|------|----------|
| `donghao.go` | httpPost 添加 token 存在性检查 |
| `donghao.go` | heartbeatParams.TokenID → Token |

### v1.3 (2026-04-10)

#### 重大变更：统一使用 token 参数
- **移除 tokenid 参数，统一使用 token**
  - 所有 API 接口参数名从 `tokenid` 改为 `token`
  - 服务端所有模块（login/heartbeat/logout/profiler/constant）统一使用 `token` 字段
  - `GetTokenID()` 方法只读取 `token` 字段，不再兼容旧版 `tokenid`

#### 变更文件
| 文件 | 变更内容 |
|------|----------|
| `login.php` | 返回字段 `tokenid` → `token` |
| `heartbeat.php` | 接收参数 `tokenid` → `token` |
| `logout.php` | 接收参数 `tokenid` → `token` |
| `profiler.php` | 接收参数 `tokenid` → `token` |
| `constant.php` | 接收参数 `tokenid` → `token` |
| `donghao.go` | 所有函数参数和请求参数改为 `token` |

### v1.2 (2026-04-10)

#### 重大变更：Token 认证机制升级
- **服务端改用随机 Token 替代数据库自增 ID**
  - 登录时服务端生成随机 Token（MD5 格式）并存储到 `heartbeat.token` 字段
  - 所有会话验证（心跳、注销、取常量等）改用 `token` 字段查询
  - 客户端无需修改，SDK 已正确处理 Token 轮换机制

#### 服务端变更
| 文件 | 变更内容 |
|------|----------|
| `login.php` | 生成随机 Token 并返回 `token` |
| `heartbeat.php` | 使用 `token` 字段查询 |
| `logout.php` | 使用 `token` 字段查询 |
| `profiler.php` | 使用 `token` 字段查询 |
| `constant.php` | 使用 `token` 字段查询 |
| `other.php` | 添加 URL 解码功能 |

#### 认证流程变化
```
之前: 登录返回数据库自增ID → 心跳使用ID查询
现在: 登录生成随机Token → 心跳使用Token查询
```

#### 优势
- 更安全：Token 随机生成，不易被猜测
- 更灵活：Token 可随时更换，不依赖数据库 ID
- 更兼容：Token 格式统一，便于跨系统集成

### v1.1 (2026-04-10)

#### Bug 修复
- **修复登录后心跳失败的问题**
  - 问题原因：SDK 在解析响应时直接使用了顶层的 `token`（MD5 hash），而没有优先使用 `result.token`
  - 影响范围：所有加密模式（RC4/RSA/Base64/AES-GCM）下的登录和心跳功能
  - 修复方案：修改 `httpPost` 函数中的 Token 设置逻辑，优先使用 `result.token`

#### 代码变更
- `donghao.go`: 修改 `httpPost` 函数中三处 Token 设置逻辑
  ```go
  // 修复前
  if result.Token != "" {
      c.currentToken = result.Token
  }
  
  // 修复后
  if token := result.GetToken(); token != "" {
      c.currentToken = token
  } else if result.Token != "" {
      c.currentToken = result.Token
  }
  ```

### v1.0 (2026-03-31)
- 初始版本发布
- 完整的 API 封装
- 支持多种加密方式
- 设备信息采集功能

---

## 功能特性

- **完整的 API 封装** - 用户管理、卡密管理、云变量、云计算函数、黑名单等全部接口
- **多种加密方式** - RC4 / RSA / Base64 / AES-256-GCM / 自定义加密
- **MD5 签名机制** - 确保请求安全性和数据完整性
- **设备信息采集** - 机器码、硬件 ID 获取，支持 Windows/Android
- **自动 Token 轮换** - 服务端返回新 Token 时自动更新
- **随机 Token 认证** - 更安全的会话管理机制

---

## 安装

```bash
go get github.com/yuan71058/DONGHAO-GO-SDK
```

**环境要求**: Go 1.25+

---

## 快速开始

### 基础用法

```go
package main

import (
    "fmt"
    "log"
    "github.com/yuan71058/DONGHAO-GO-SDK"
)

func main() {
    // 创建客户端
    client := donghao.NewClient("https://your-api-domain.com", 1)
    
    // 获取设备信息
    mac := donghao.GetMachineCodeSafe()
    ip := donghao.GetLocalIP()
    clientID := donghao.GenerateClientID()
    
    // 用户登录
    result, err := client.Login("username", "password", "1.0", mac, ip, clientID)
    if err != nil {
        log.Fatal(err)
    }
    
    if result.IsSuccess() {
        fmt.Println("登录成功，Token:", client.GetToken())
        
        // 启动自动心跳
        client.StartAutoHeartbeat("", "", "1.0", mac, ip, clientID)
        defer client.StopAutoHeartbeat()
        
        // 获取用户信息
        userResult, _ := client.GetUser("username", client.GetToken(), "1.0", mac, ip, clientID)
        fmt.Println("用户信息:", userResult.Msg())
    } else {
        fmt.Println("登录失败:", result.Msg())
    }
}
```

### 启用加密

```go
client := donghao.NewClient("https://your-api-domain.com", 1)

// RC4 加密
client.SetEncryption(donghao.ENC_RC4, "your-rc4-key")

// AES-256-GCM 加密（推荐）
client.SetEncryption(donghao.ENC_AES_GCM, "your-32-byte-key-here!!!")

// RSA 加密
client.SetEncryption(donghao.ENC_RSA, "your-public-key")
```

### 启用签名

```go
client.SetSignConfig("your-app-key", "[data]xxx[key]yyy", true)
```

---

## API 接口列表

### Result 对象方法

```go
type Result struct {
    Code   int         // 返回状态码
    Result interface{} // 返回数据
    Data   interface{} // 兼容字段
    Uuid   string      // 客户端UUID
    Token  string      // 新 Token
}
```

| 方法 | 说明 |
|------|------|
| `IsSuccess() bool` | 判断是否成功 |
| `Msg() string` | 获取返回消息 |
| `GetToken() string` | 获取 Token（随机 Token） |
| `GetData() (string, error)` | 获取用户数据（Base64 解码） |
| `GetGroupData() (string, error)` | 获取分组数据（Base64 解码） |
| `GetVariableValue() (string, error)` | 获取变量值（Base64 解码） |

### 用户管理

| 方法 | 说明 |
|------|------|
| `Login(user, pwd, ver, mac, ip, clientid)` | 用户登录 |
| `LoginCard(card, ver, mac, ip, clientid)` | 卡密登录 |
| `Reg(user, pwd, card, qq, email, tjr, ver, mac, ip, clientid)` | 用户注册 |
| `Logout(user, token, ver, mac, ip, clientid)` | 用户注销 |
| `Heartbeat(user, token, ver, mac, ip, clientid)` | 心跳维持在线 |
| `GetUser(user, token, ver, mac, ip, clientid)` | 获取用户信息 |
| `Uppwd(user, pwd, newpwd, ver, mac, ip, clientid)` | 修改密码 |
| `Binding(user, pwd, newuser, newmac, newip, qq, ver, mac, ip, clientid)` | 绑定新设备 |
| `Bindreferrer(user, pwd, tjr, ver, mac, ip, clientid)` | 绑定推荐人 |

### 卡密管理

| 方法 | 说明 |
|------|------|
| `Recharge(user, card, ver, mac, ip, clientid)` | 卡密充值 |

### 用户数据

| 方法 | 说明 |
|------|------|
| `GetUdata(user, token, ver, mac, ip, clientid)` | 获取用户数据 |
| `SetUdata(user, token, udata, ver, mac, ip, clientid)` | 设置用户数据 |
| `GetUdata2(user, token, ver, mac, ip, clientid)` | 获取用户数据2 |
| `SetUdata2(user, token, udata, ver, mac, ip, clientid)` | 设置用户数据2 |

### 云变量/常量

| 方法 | 说明 |
|------|------|
| `GetVariable(user, token, key, ver, mac, ip, clientid)` | 获取云变量 |
| `SetVariable(user, token, key, value, ver, mac, ip, clientid)` | 设置云变量 |
| `DelVariable(user, token, key, ver, mac, ip, clientid)` | 删除云变量 |
| `Constant(user, token, key, ver, mac, ip, clientid)` | 获取云常量 |

### 云计算函数

| 方法 | 说明 |
|------|------|
| `Func(user, token, func, para, ver, mac, ip, clientid)` | 云计算函数（需登录） |
| `Func2(func, para, ver, mac, ip, clientid)` | 云计算函数（免登录） |
| `CallPHP(user, token, func, para, ver, mac, ip, clientid)` | 调用 PHP 函数（需登录） |
| `CallPHP2(func, para, ver, mac, ip, clientid)` | 调用 PHP 函数（免登录） |

### 黑名单管理

| 方法 | 说明 |
|------|------|
| `GetBlack(bType, bData)` | 查询黑名单 (type: ip/mac/user) |
| `SetBlack(bType, data, note, ver, mac, ip, clientid)` | 添加黑名单 |

### 其他接口

| 方法 | 说明 |
|------|------|
| `DeductPoints(user, token, count, ver, mac, ip, clientid)` | 扣除积分 |
| `AddLog(user, info, ver, mac, ip, clientid)` | 添加日志 |
| `CheckAuth(user, pwd, md5, ver, mac, ip, clientid)` | 验证授权 |
| `Init(ver, mac, ip, clientid)` | 软件初始化 |
| `Notice(title, ver, mac, ip, clientid)` | 获取公告 |
| `Ver(ver, mac, ip, clientid)` | 版本检查 |
| `Relay(params, ver, mac, ip, clientid)` | 中继转发 |

---

## 错误处理

```go
result, err := client.Login(user, pwd, ver, mac, ip, clientid)

if err != nil {
    fmt.Println("网络错误:", err)
    return
}

if result.IsSuccess() {
    fmt.Println("成功")
} else {
    fmt.Printf("失败: code=%d, msg=%s\n", result.Code, result.Msg())
}
```

| 错误码 | 含义 |
|--------|------|
| 200 / 1 | 成功 |
| 其他值 | 业务错误（具体含义参考服务端文档） |

---

## 自动心跳

```go
// 启动自动心跳（后台 goroutine 定时发送）
client.StartAutoHeartbeat("", "", "1.0", mac, ip, clientID)

// 带错误回调的自动心跳
client.StartAutoHeartbeatWithCallback("", "", "1.0", mac, ip, clientID, 
    func(err error, failures int) {
        fmt.Printf("心跳失败 %d 次: %v\n", failures, err)
    })

// 停止自动心跳
defer client.StopAutoHeartbeat()
```

---

## 加密类型

```go
const (
    ENC_NONE    = 0 // 不加密
    ENC_RC4     = 1 // RC4 流加密
    ENC_RSA     = 2 // RSA 非对称加密
    ENC_BASE64  = 3 // Base64 编码
    ENC_CUSTOM  = 4 // 自定义加密
    ENC_AES_GCM = 5 // AES-256-GCM 认证加密
)
```

---

## 设备信息采集

```go
// 获取机器码（Windows: CPU+主板+硬盘序列号组合）
mac := donghao.GetMachineCodeSafe()

// 获取本地 IP 地址
ip := donghao.GetLocalIP()

// 生成唯一客户端 ID
clientID := donghao.GenerateClientID()

// 获取硬件信息（详细）
hwInfo, _ := donghao.GetHardwareInfo()
fmt.Println(hwInfo.CPUID, hwInfo.BaseboardSerial, hwInfo.DiskSerial)
```

---

## 注意事项

1. **Token 管理**: 登录成功后 SDK 自动保存 Token，后续调用会自动使用
2. **心跳维持**: 建议启动自动心跳防止会话过期
3. **加密选择**: 生产环境建议使用 AES-256-GCM 或 RSA
4. **错误处理**: 始终检查 `err` 和 `result.IsSuccess()`
5. **线程安全**: 同一客户端实例可并发使用

---

## 许可证

MIT License

---

## 相关链接

- [冬浩验证系统](https://github.com/yuan71058/donghao)
- [Go SDK GitHub](https://github.com/yuan71058/DONGHAO-GO-SDK)
- [API 详细文档](API.md)
