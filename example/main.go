// Package main 冬浩验证系统Go SDK使用示例
//
// 本示例展示了如何使用冬浩验证系统Go SDK进行各种操作，包括：
//   - 用户登录、注册、注销
//   - 卡密充值和绑定
//   - 云变量、常量操作
//   - 用户数据管理
//   - 心跳维持在线状态
//   - 黑名单管理等
//
// 运行方式:
//
//	go run example/main.go
//
// 版本: 1.4
// 作者: 冬浩验证系统
// 日期: 2026-04-11
package main

import (
	"fmt"
	"log"
	"time"

	donghao "github.com/yuan71058/DONGHAO-GO-SDK"
)

func main() {
	fmt.Println("=== 冬浩验证系统 - Go语言SDK 完整使用示例 ===\n")

	// 首先获取设备信息（机器码等）
	exampleDeviceID()

	// 创建客户端实例
	client := donghao.NewClient("http://your-domain.com", 1)
	client.SetTimeout(30)

	// 按顺序执行各种功能示例
	exampleInit(client)
	exampleLogin(client)
	exampleLoginCard(client) // 卡密登录示例
	exampleRegister(client)
	exampleHeartbeat(client)
	exampleGetUser(client)
	exampleUserData(client)
	exampleVariable(client)
	exampleConstant(client)
	exampleBlacklist(client)
	exampleOtherFeatures(client)
	exampleCloudFunctions(client)
	// exampleAutoHeartbeat(client) // 自动心跳维持在线功能（基于context.Context）
	// exampleFullFlow()            // 完整实战流程（卡密登录 + 自动心跳）
}

// exampleDeviceID 展示设备信息采集功能
//
// 设备信息函数包括：
//   - GetMachineCode(): 获取当前设备的唯一机器码（可能出错）
//   - GetMachineCodeSafe(): 备用方法获取机器码（不会报错）
//   - GetHardwareID(): 获取当前设备的唯一硬件ID
//   - GetHardwareInfo(): 获取当前设备的详细硬件信息
//   - GenerateClientID(): 生成随机客户端ID
//
// 机器码生成算法说明：
//
//	Windows: 硬件信息(CPU+主板+磁盘+MAC+BIOS) -> MD5 -> 格式化(XXXX-XXXX-XXXX-XXXX)
//	Android: 设备信息 -> MD5 -> 格式化(XXXX-XXXX-XXXX-XXXX)
func exampleDeviceID() {
	fmt.Println("1. 设备信息采集功能演示")
	fmt.Println("----------------------------------------")

	// 获取当前设备的唯一机器码（可能出错）
	machineCode, err := donghao.GetMachineCode()
	if err != nil {
		fmt.Printf("获取机器码失败: %v\n", err)
		// 如果获取失败，使用备用方法生成机器码
		machineCode = donghao.GetMachineCodeSafe()
		fmt.Printf("备用方法机器码: %s\n", machineCode)
	} else {
		fmt.Printf("机器码: %s\n", machineCode)
	}

	// 获取当前设备的硬件ID
	hardwareID, err := donghao.GetHardwareID()
	if err != nil {
		fmt.Printf("获取硬件ID失败: %v\n", err)
	} else {
		fmt.Printf("硬件ID: %s\n", hardwareID)
	}

	// 获取当前设备的详细硬件信息
	hwInfo, err := donghao.GetHardwareInfo()
	if err != nil {
		fmt.Printf("获取硬件详细信息失败: %v\n", err)
	} else {
		fmt.Println("硬件详细信息:")
		for key, value := range hwInfo {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	// 生成随机客户端ID，用于标识不同设备会话
	clientID := donghao.GenerateClientID()
	fmt.Printf("客户端ID: %s\n", clientID)

	fmt.Println()
}

// exampleInit 展示软件初始化功能
//
// 初始化功能包括：
//  1. 调用client.Init进行软件初始化
//  2. 解析返回的初始化信息
//
// 初始化返回的信息包括：
//   - name: 软件名称
//   - orcheck: 运行检查结果(0=正常 1=需绑定 2=需更新 3=已过期)
//   - xttime: 到期时间戳
//   - notice: 软件公告
//   - data: 附加数据(base64编码)
func exampleInit(client *donghao.Client) {
	fmt.Println("2. 软件初始化功能演示")
	fmt.Println("----------------------------------------")

	result, err := client.Init(
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("初始化错误: %v\n", err)
		fmt.Println()
		return
	}

	if result.IsSuccess() {
		fmt.Printf("初始化成功!\n")
		m := result.GetResultMap()
		if m != nil {
			if name, ok := m["name"].(string); ok {
				fmt.Printf("软件名称: %s\n", name)
			}
			if orcheck, ok := m["orcheck"].(float64); ok {
				modes := []string{"正常", "需绑定", "需更新", "已过期"}
				mode := int(orcheck)
				if mode >= 0 && mode < len(modes) {
					fmt.Printf("运行检查结果: %s\n", modes[mode])
				}
			}
			if notice, ok := m["notice"].(string); ok {
				fmt.Printf("软件公告: %s\n", notice)
			}
			if xttime, ok := m["xttime"].(float64); ok {
				fmt.Printf("到期时间: %.0f\n", xttime)
			}
		}

		data, decodeErr := result.GetData()
		if decodeErr == nil && data != "" {
			fmt.Printf("附加数据: %s\n", data)
		}
	} else {
		fmt.Printf("初始化失败: %s\n", result.Msg())
	}

	fmt.Println()
}

// exampleLogin 展示用户登录功能
//
// 登录流程包括：
//  1. 调用client.Login进行登录
//  2. 解析返回的结果
//
// 登录成功后可以：
//   - Token字符串可通过client.GetToken()获取
//   - 当前用户名可通过client.GetCurrentUser()获取
func exampleLogin(client *donghao.Client) {
	fmt.Println("3. 用户登录功能演示")
	fmt.Println("----------------------------------------")

	result, err := client.Login(
		"testuser",
		"password123",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("登录错误: %v\n", err)
		fmt.Println()
		return
	}

	if result.IsSuccess() {
		fmt.Printf("登录成功!\n")
		fmt.Printf("Token: %s\n", client.GetToken())
		fmt.Printf("当前用户: %s\n", client.GetCurrentUser())
		fmt.Printf("返回数据: %v\n", result.Data)
	} else {
		fmt.Printf("登录失败: %s\n", result.Msg())
	}

	fmt.Println()
}

// exampleLoginCard 展示卡密登录功能
//
// 卡密登录是一种无需账号密码的登录方式：
//   - 使用充值卡号直接登录并创建/关联账号
//   - 需要后台开启"充值卡登录模式"(dl_type=1)
//   - 适用于发卡平台销售场景
//
// 参数说明：
//   - card: 卡密号 (必填)
//   - ver: 软件版本号 (必填)
//   - mac: 设备MAC地址 (必填)
//   - ip: IP地址 (必填)
//   - clientid: 客户端ID (必填)
func exampleLoginCard(client *donghao.Client) {
	fmt.Println("3.5 卡密登录功能演示")
	fmt.Println("----------------------------------------")

	result, err := client.LoginCard(
		"CARD-TEST-123456",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("卡密登录错误: %v\n", err)
		fmt.Println()
		return
	}

	if result.IsSuccess() {
		fmt.Printf("卡密登录成功!\n")
		fmt.Printf("Token: %s\n", client.GetToken())
		fmt.Printf("当前用户: %s\n", client.GetCurrentUser())
	} else {
		fmt.Printf("卡密登录失败: %s\n", result.Msg())
	}

	fmt.Println()
}

// exampleRegister 展示用户注册功能
//
// 注册需要填写的信息包括：
//   - user: 用户名 (必填)
//   - pwd: 密码 (必填)
//   - card: 注册所需卡密(可选)
//   - userqq: QQ号 (可选)
//   - email: 电子邮箱地址(可选)
//   - tjr: 推荐人用户名(可选)
//   - ver: 软件版本号 (必填)
//   - mac: 设备MAC地址 (必填)
//   - ip: IP地址 (必填)
//   - clientid: 客户端ID (必填)
func exampleRegister(client *donghao.Client) {
	fmt.Println("4. 用户注册功能演示")
	fmt.Println("----------------------------------------")

	result, err := client.Reg(
		"newuser123",
		"password123",
		"",
		"123456",
		"test@example.com",
		"",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("注册错误: %v\n", err)
		fmt.Println()
		return
	}

	if result.IsSuccess() {
		fmt.Printf("注册成功: %v\n", result.Data)
	} else {
		fmt.Printf("注册失败: %s\n", result.Msg())
	}

	fmt.Println()
}

// exampleHeartbeat 展示心跳维持在线功能
//
// 心跳维持的作用：
//   - 定期向服务器发送心跳包以保持账号在线状态
//   - 建议间隔时间根据业务需求设置，通常为60-300秒
//   - 心跳失败会导致用户自动下线
func exampleHeartbeat(client *donghao.Client) {
	fmt.Println("5. 心跳维持在线功能演示")
	fmt.Println("----------------------------------------")

	result, err := client.Heartbeat(
		"testuser",
		"token123",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("心跳错误: %v\n", err)
		fmt.Println()
		return
	}

	if result.IsSuccess() {
		fmt.Printf("心跳成功!\n")
		fmt.Printf("返回数据: %v\n", result.Data)
	} else {
		fmt.Printf("心跳失败: %s\n", result.Msg())
	}

	fmt.Println()
}

// exampleGetUser 展示获取用户信息功能
//
// 该功能用于获取用户的详细账户信息，
//
//	包括用户名、到期时间、会员等级等信息
func exampleGetUser(client *donghao.Client) {
	fmt.Println("6. 获取用户信息功能演示")
	fmt.Println("----------------------------------------")

	result, err := client.GetUser(
		"testuser",
		"token123",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("获取用户信息错误: %v\n", err)
		fmt.Println()
		return
	}

	if result.IsSuccess() {
		fmt.Printf("获取用户信息成功!\n")
		fmt.Printf("返回数据: %v\n", result.Data)
	} else {
		fmt.Printf("获取失败: %s\n", result.Msg())
	}

	fmt.Println()
}

// exampleUserData 展示用户数据存储功能
//
// 用户数据功能包括：
//   - 存储自定义的用户配置或游戏进度等数据
//   - 数据会经过Base64编码后存储在服务器上
//   - 支持两组独立的数据存储空间
func exampleUserData(client *donghao.Client) {
	fmt.Println("7. 用户数据存储功能演示")
	fmt.Println("----------------------------------------")

	fmt.Println("7.1 设置用户数据")
	data := `{"settings": {"theme": "dark", "language": "zh-CN"}}`

	result, err := client.SetUdata(
		"testuser",
		"token123",
		data,
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("设置用户数据错误: %v\n", err)
	} else if result.IsSuccess() {
		fmt.Printf("设置数据成功: %v\n", result.Data)
	} else {
		fmt.Printf("设置失败: %s\n", result.Msg())
	}

	fmt.Println("\n7.2 获取用户数据")
	userData, err := client.GetUdata(
		"testuser",
		"token123",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("获取用户数据错误: %v\n", err)
	} else if userData.IsSuccess() {
		fmt.Printf("用户数据: %v\n", userData.Data)
	} else {
		fmt.Printf("获取失败: %s\n", userData.Msg())
	}

	fmt.Println()
}

// exampleVariable 展示云变量操作功能
//
// 云变量的作用：
//   - 允许服务端动态控制客户端行为
//   - 可以通过后台修改云变量值来实时更新客户端配置
func exampleVariable(client *donghao.Client) {
	fmt.Println("8. 云变量操作功能演示")
	fmt.Println("----------------------------------------")

	fmt.Println("8.1 设置云变量")
	varData := `{"key": "value", "config": "test"}`

	result, err := client.SetVariable(
		"testuser",
		"token123",
		"config",
		varData,
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("设置云变量错误: %v\n", err)
	} else if result.IsSuccess() {
		fmt.Printf("设置成功: %v\n", result.Data)
	} else {
		fmt.Printf("设置失败: %s\n", result.Msg())
	}

	fmt.Println("\n8.2 获取云变量")
	value, err := client.GetVariable(
		"testuser",
		"token123",
		"config",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("获取云变量错误: %v\n", err)
	} else if value.IsSuccess() {
		varValue, decodeErr := value.GetVariableValue()
		if decodeErr != nil {
			fmt.Printf("解码错误: %v\n", decodeErr)
			fmt.Printf("原始数据: %v\n", value.GetResultMap())
		} else {
			fmt.Printf("变量值: %s\n", varValue)
		}
	} else {
		fmt.Printf("获取失败: %s\n", value.Msg())
	}

	fmt.Println("\n8.3 删除云变量")
	result2, err := client.DelVariable(
		"testuser",
		"token123",
		"config",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("删除云变量错误: %v\n", err)
	} else if result2.IsSuccess() {
		fmt.Printf("删除成功: %v\n", result2.Data)
	} else {
		fmt.Printf("删除失败: %s\n", result2.Msg())
	}

	fmt.Println()
}

// exampleConstant 展示获取云常量功能
//
// 常量的特点：
//   - 常量只能由后台设置，客户端只能读取
//   - 适用于固定不变的配置项
func exampleConstant(client *donghao.Client) {
	fmt.Println("9. 获取云常量功能演示")
	fmt.Println("----------------------------------------")

	result, err := client.Constant(
		"testuser",
		"token123",
		"test_key",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("获取常量错误: %v\n", err)
	} else if result.IsSuccess() {
		fmt.Printf("常量值: %v\n", result.Data)
	} else {
		fmt.Printf("获取失败: %s\n", result.Msg())
	}

	fmt.Println()
}

// exampleBlacklist 展示黑名单管理功能
//
// 黑名单功能的应用场景：
//   - 可以将IP地址/MAC地址/用户名加入黑名单来禁止访问
//   - 可以查询某个IP/MAC/用户是否已被封禁
func exampleBlacklist(client *donghao.Client) {
	fmt.Println("10. 黑名单管理功能演示")
	fmt.Println("----------------------------------------")

	fmt.Println("10.1 查询黑名单")
	result, err := client.GetBlack(
		"ip",
		"192.168.1.100",
	)

	if err != nil {
		log.Printf("查询黑名单错误: %v\n", err)
	} else if result.IsSuccess() {
		fmt.Printf("黑名单结果: %v\n", result.Data)
	} else {
		fmt.Printf("查询失败: %s\n", result.Msg())
	}

	fmt.Println("\n10.2 设置黑名单")
	result2, err := client.SetBlack(
		"ip",
		"192.168.1.100",
		"违规操作被封禁",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("设置黑名单错误: %v\n", err)
	} else if result2.IsSuccess() {
		fmt.Printf("设置成功: %v\n", result2.Data)
	} else {
		fmt.Printf("设置失败: %s\n", result2.Msg())
	}

	fmt.Println()
}

// exampleOtherFeatures 展示其他辅助功能
//
// 包括的功能有：
//   - 卡密充值
//   - 修改密码
//   - 绑定新设备（支持更换用户名/MAC/IP/QQ等）
//   - 绑定推荐人
//   - 扣除积分
//   - 添加日志
//   - 用户注销
func exampleOtherFeatures(client *donghao.Client) {
	fmt.Println("11. 其他辅助功能演示")
	fmt.Println("----------------------------------------")

	fmt.Println("11.1 卡密充值")
	result, err := client.Recharge(
		"testuser",
		"CARD-123456",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("充值错误: %v\n", err)
	} else if result.IsSuccess() {
		fmt.Printf("充值成功: %v\n", result.Data)
	} else {
		fmt.Printf("充值失败: %s\n", result.Msg())
	}

	fmt.Println("\n11.2 修改密码")
	result2, err := client.Uppwd(
		"testuser",
		"old_password",
		"new_password",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("修改密码错误: %v\n", err)
	} else if result2.IsSuccess() {
		fmt.Printf("修改密码成功: %v\n", result2.Data)
	} else {
		fmt.Printf("修改密码失败: %s\n", result2.Msg())
	}

	fmt.Println("\n11.3 绑定新设备（支持更换用户名/MAC/IP/QQ等）")
	result3, err := client.Binding(
		"testuser",
		"password",
		"newusername",       // 如果要更改用户名则填写新用户名，否则留空
		"AA:BB:CC:DD:EE:FF", // 如果要更换设备则填写新MAC地址，否则留空
		"192.168.1.200",     // 如果要更换设备则填写新IP地址，否则留空
		"123456789",         // 如果要更换则填写新QQ号，否则留空
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("绑定错误: %v\n", err)
	} else if result3.IsSuccess() {
		fmt.Printf("绑定成功: %v\n", result3.Data)
	} else {
		fmt.Printf("绑定失败: %s\n", result3.Msg())
	}

	fmt.Println("\n11.4 绑定推荐人")
	result4, err := client.Bindreferrer(
		"testuser",
		"password",
		"referrer_user",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("绑定推荐人错误: %v\n", err)
	} else if result4.IsSuccess() {
		fmt.Printf("绑定推荐人成功: %v\n", result4.Data)
	} else {
		fmt.Printf("绑定推荐人失败: %s\n", result4.Msg())
	}

	fmt.Println("\n11.5 扣除积分")
	result5, err := client.DeductPoints(
		"testuser",
		"token123",
		10,
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("扣除积分错误: %v\n", err)
	} else if result5.IsSuccess() {
		fmt.Printf("扣除积分成功: %v\n", result5.Data)
	} else {
		fmt.Printf("扣除积分失败: %s\n", result5.Msg())
	}

	fmt.Println("\n11.6 添加日志")
	result6, err := client.AddLog(
		"testuser",
		"[用户日志] 测试添加一条用户操作日志到服务器",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("添加日志错误: %v\n", err)
	} else if result6.IsSuccess() {
		fmt.Printf("添加日志成功: %v\n", result6.Data)
	} else {
		fmt.Printf("添加日志失败: %s\n", result6.Msg())
	}

	fmt.Println("\n11.7 验证授权（无需密码）")
	result7, err := client.CheckAuth(
		"testuser",
		"password",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
		"",
	)

	if err != nil {
		log.Printf("验证授权错误: %v\n", err)
	} else if result7.IsSuccess() {
		fmt.Printf("验证授权成功: %v\n", result7.Data)
	} else {
		fmt.Printf("验证授权失败: %s\n", result7.Msg())
	}

	fmt.Println()
}

// exampleCloudFunctions 展示云计算函数调用功能
//
// 云计算函数的作用：
//   - 在服务端执行预定义的计算逻辑
//   - 支持带参数的函数调用
//   - 可用于实现复杂的业务逻辑
func exampleCloudFunctions(client *donghao.Client) {
	fmt.Println("12. 云计算函数调用功能演示")
	fmt.Println("----------------------------------------")

	fmt.Println("12.1 带用户信息的云计算函数调用")
	result, err := client.Func(
		"testuser",
		"token123",
		"jia",
		"1,2",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("云计算函数错误: %v\n", err)
	} else if result.IsSuccess() {
		fmt.Printf("计算结果: %v\n", result.Data)
	} else {
		fmt.Printf("计算失败: %s\n", result.Msg())
	}

	fmt.Println("\n12.2 免登录云计算函数调用")
	result2, err := client.Func2(
		"jia",
		"1,2",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("免登录计算错误: %v\n", err)
	} else if result2.IsSuccess() {
		fmt.Printf("免登录计算结果: %v\n", result2.Data)
	} else {
		fmt.Printf("免登录计算失败: %s\n", result2.Msg())
	}

	fmt.Println("\n12.3 调用PHP函数（需要登录）")
	result3, err := client.CallPHP(
		"testuser",
		"token123",
		"test_func",
		"param1,param2",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("调用PHP函数错误: %v\n", err)
	} else if result3.IsSuccess() {
		fmt.Printf("PHP函数结果: %v\n", result3.Data)
	} else {
		fmt.Printf("PHP函数调用失败: %s\n", result3.Msg())
	}

	fmt.Println("\n12.4 调用PHP函数（无需登录）")
	result4, err := client.CallPHP2(
		"test_func",
		"param1,param2",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("免登录PHP调用错误: %v\n", err)
	} else if result4.IsSuccess() {
		fmt.Printf("免登录PHP结果: %v\n", result4.Data)
	} else {
		fmt.Printf("免登录PHP调用失败: %s\n", result4.Msg())
	}

	fmt.Println()
}

// exampleAutoHeartbeat 展示自动心跳维持在线功能
//
// 自动心跳在后台goroutine中定时发送心跳请求，维持用户在线状态。
// 使用 context.Context 实现优雅取消，支持错误回调。
//
// 使用方式：
//  1. 登录成功后调用 StartAutoHeartbeat 启动（user/token 可为空自动填充）
//  2. 程序退出前调用 StopAutoHeartbeat 停止
//  3. 可选：使用 IsHeartbeatRunning() 检查状态
//
// Token 机制说明（v1.2+）：
//   - 登录时服务端返回随机 Token（MD5格式）
//   - 心跳时使用该 Token 进行会话验证
//   - SDK 自动管理 Token 的存储和传递，无需手动处理
func exampleAutoHeartbeat(client *donghao.Client) {
	fmt.Println("13. 自动心跳维持在线功能演示")
	fmt.Println("----------------------------------------")

	// 设置心跳间隔为10秒（演示用，实际建议60-300秒）
	client.HeartbeatInterval = 10 * time.Second

	// 方式一：基本用法（启动自动心跳，user/token 为空时自动使用登录信息）
	err := client.StartAutoHeartbeat(
		"", // user: 为空自动使用 currentUser
		"", // token: 为空自动使用 currentToken
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("启动自动心跳错误: %v\n", err)
		fmt.Println()
		return
	}

	fmt.Println("✅ 自动心跳已启动!")
	fmt.Printf("   心跳状态: %v\n", client.IsHeartbeatRunning())
	fmt.Printf("   当前Token: %s\n", client.GetToken())
	fmt.Println("   心跳将在后台自动运行...")

	// 模拟程序运行一段时间
	time.Sleep(15 * time.Second)

	fmt.Printf("   心跳状态: %v\n", client.IsHeartbeatRunning())

	// 停止自动心跳
	client.StopAutoHeartbeat()
	fmt.Println("✅ 自动心跳已停止")
	fmt.Printf("   心跳状态: %v\n", client.IsHeartbeatRunning())

	fmt.Println()

	// 方式二：带错误回调的高级用法
	// err = client.StartAutoHeartbeatWithCallback("", "", "1.0.0", mac, ip, clientID,
	// 	func(err error, count int) {
	// 		log.Printf("[心跳告警] 连续失败%d次: %v\n", count, err)
	// 		if count >= 3 {
	// 			log.Println("[心跳告警] 连续失败超过3次，可能需要重新登录！")
	// 		}
	// 	},
	// )
}

// exampleFullFlow 展示完整的卡密登录 + 自动心跳实战流程
//
// 这是最常见的使用场景：
//  1. 用户输入卡密 → 调用 LoginCard 登录
//  2. 登录成功 → 启动自动心跳维持在线
//  3. 程序退出 → 停止心跳并清理资源
//
// 适用场景：发卡平台、软件授权验证等
func exampleFullFlow() {
	fmt.Println("14. 完整实战流程演示（卡密登录 + 自动心跳）")
	fmt.Println("----------------------------------------")

	// 创建客户端
	client := donghao.NewClient("http://your-domain.com", 1)
	client.SetTimeout(30)

	// 获取设备信息
	mac := donghao.GetMachineCodeSafe()
	ip := donghao.GetLocalIP()
	clientID := donghao.GenerateClientID()

	fmt.Printf("设备信息: MAC=%s IP=%s ClientID=%s\n", mac, ip, clientID)

	// 步骤1: 卡密登录
	cardKey := "YOUR_CARD_KEY_HERE"
	result, err := client.LoginCard(cardKey, "1.0.0", mac, ip, clientID)
	if err != nil {
		log.Fatalf("❌ 卡密登录请求失败: %v\n", err)
	}

	if !result.IsSuccess() {
		log.Fatalf("❌ 卡密登录失败: %s\n", result.Msg())
	}

	fmt.Printf("✅ 卡密登录成功!\n")
	fmt.Printf("   用户名: %s\n", client.GetCurrentUser())
	fmt.Printf("   Token: %s\n", client.GetToken())

	if m := result.GetResultMap(); m != nil {
		if endtime, ok := m["endtime"].(string); ok {
			fmt.Printf("   到期时间: %s\n", endtime)
		}
		if point, ok := m["point"].(float64); ok {
			fmt.Printf("   剩余点数: %.0f\n", point)
		}
	}

	// 步骤2: 启动自动心跳（user和token为空，自动使用登录后的值）
	err = client.StartAutoHeartbeatWithCallback(
		"", // user: 自动使用 currentUser
		"", // token: 自动使用 currentToken
		"1.0.0",
		mac,
		ip,
		clientID,
		func(err error, count int) {
			log.Printf("⚠️ 心跳连续失败(%d次): %v\n", count, err)
			if count >= 3 {
				log.Println("🔴 连续失败超过3次，建议重新登录！")
			}
		},
	)
	if err != nil {
		log.Fatalf("❌ 启动自动心跳失败: %v\n", err)
	}

	fmt.Println("✅ 自动心跳已启动，程序开始正常运行...")
	fmt.Println("   (按 Ctrl+C 或等待15秒后自动停止)")

	// 步骤3: 模拟程序主循环运行
	time.Sleep(15 * time.Second)

	// 步骤4: 程序退出前停止心跳
	client.StopAutoHeartbeat()
	fmt.Println("✅ 自动心跳已停止，程序正常退出")

	fmt.Println()
}
