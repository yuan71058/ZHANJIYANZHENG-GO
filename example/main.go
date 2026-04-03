// Package main 提供冬浩验证系统Go SDK的使用示例
//
// 本文件演示如何在Go项目中使用冬浩验证SDK。
// 包含用户登录注册、心跳检测、数据存取、变量管理、
// 充值卡使用、云计算等功能的完整示例。
//
// 运行方法:
//
//	go run example/main.go
//
// 版本: 1.0
// 作者: 冬浩验证系统
// 日期: 2026-03-31
package main

import (
	"fmt"
	"log"

	donghao "gitee.com/yuan71058/dong-hao-verification"
)

func main() {
	fmt.Println("=== 冬浩验证系统 - Go语言调用示例 ===\n")

	// 演示设备标识获取（在创建客户端之前）
	exampleDeviceID()

	// 创建客户端
	client := donghao.NewClient("http://your-domain.com", 1)
	client.SetTimeout(30)

	// 运行各个示例
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
	// exampleAutoHeartbeat(client) // 已禁用：自动心跳功能暂不可用
}

// exampleDeviceID 演示设备标识获取
//
// 设备标识说明:
//   - GetMachineCode(): 获取基于硬件的机器码（跨平台）
//   - GetMachineCodeSafe(): 安全获取机器码（不会失败）
//   - GetHardwareID(): 获取详细硬件ID
//   - GetHardwareInfo(): 获取详细硬件信息
//   - GenerateClientID(): 生成客户端ID
//
// 机器码生成采用双层MD5加密:
//
//	第一层: 硬件信息 → MD5 → 中间码
//	第二层: 中间码 → MD5 → 最终机器码
func exampleDeviceID() {
	fmt.Println("【示例】设备标识获取")
	fmt.Println("----------------------------------------")

	// 获取机器码（基于真实硬件信息）
	machineCode, err := donghao.GetMachineCode()
	if err != nil {
		fmt.Printf("获取机器码失败: %v\n", err)
		// 使用安全获取方式（不会失败）
		machineCode = donghao.GetMachineCodeSafe()
		fmt.Printf("使用备用机器码: %s\n", machineCode)
	} else {
		fmt.Printf("机器码: %s\n", machineCode)
	}

	// 获取硬件ID
	hardwareID, err := donghao.GetHardwareID()
	if err != nil {
		fmt.Printf("获取硬件ID失败: %v\n", err)
	} else {
		fmt.Printf("硬件ID: %s\n", hardwareID)
	}

	// 获取详细硬件信息
	hwInfo, err := donghao.GetHardwareInfo()
	if err != nil {
		fmt.Printf("获取硬件信息失败: %v\n", err)
	} else {
		fmt.Println("硬件信息:")
		for key, value := range hwInfo {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	// 生成客户端ID（基于时间戳和随机数）
	clientID := donghao.GenerateClientID()
	fmt.Printf("客户端ID: %s\n", clientID)

	fmt.Println()
}

// exampleInit 演示软件初始化
//
// 初始化流程:
//  1. 调用client.Init方法获取软件信息
//  2. 检查返回结果，获取软件配置
//
// 初始化返回信息:
//   - name: 软件名称
//   - orcheck: 运营模式(0停运 1收费 2点数 3免费)
//   - xttime: 心跳时间
//   - notice: 软件公告
//   - data: 静态数据(base64编码)
func exampleInit(client *donghao.Client) {
	fmt.Println("【示例0】软件初始化")
	fmt.Println("----------------------------------------")

	result, err := client.Init(
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("初始化失败: %v\n", err)
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
				modes := []string{"停运", "收费", "点数", "免费"}
				mode := int(orcheck)
				if mode >= 0 && mode < len(modes) {
					fmt.Printf("运营模式: %s\n", modes[mode])
				}
			}
			if notice, ok := m["notice"].(string); ok {
				fmt.Printf("软件公告: %s\n", notice)
			}
			if xttime, ok := m["xttime"].(float64); ok {
				fmt.Printf("心跳时间: %.0f秒\n", xttime)
			}
		}

		data, decodeErr := result.GetData()
		if decodeErr == nil && data != "" {
			fmt.Printf("静态数据: %s\n", data)
		}
	} else {
		fmt.Printf("初始化失败: %s\n", result.Msg())
	}

	fmt.Println()
}

// exampleLogin 演示用户登录
//
// 登录流程:
//  1. 调用client.Login方法
//  2. 检查返回结果
//
// 登录成功后:
//   - Token会自动保存在客户端
//   - 可通过client.GetToken()获取
func exampleLogin(client *donghao.Client) {
	fmt.Println("【示例1】用户登录")
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
		log.Printf("登录失败: %v\n", err)
		fmt.Println()
		return
	}

	if result.IsSuccess() {
		fmt.Printf("登录成功!\n")
		fmt.Printf("Token: %s\n", client.GetToken())
		fmt.Printf("用户名: %s\n", client.GetCurrentUser())
		fmt.Printf("返回数据: %v\n", result.Data)
	} else {
		fmt.Printf("登录失败: %s\n", result.Msg())
	}

	fmt.Println()
}

// exampleRegister 演示用户注册
//
// 注册参数说明:
//   - user: 用户名(必填)
//   - pwd: 密码(必填)
//   - card: 注册卡密(可选)
//   - userqq: QQ号(可选)
//   - email: 邮箱地址(可选)
//   - tjr: 推荐人用户名(可选)
//   - ver: 客户端版本号(必填)
//   - mac: 设备MAC地址(必填)
//   - ip: 设备IP地址(必填)
//   - clientid: 客户端ID(必填)
func exampleRegister(client *donghao.Client) {
	fmt.Println("【示例2】用户注册")
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
		log.Printf("注册失败: %v\n", err)
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

// exampleHeartbeat 演示心跳检测
//
// 心跳机制说明:
//   - 定时向服务器发送心跳请求，保持在线状态
//   - 建议心跳间隔根据服务器配置(通常60-300秒)
//   - 心跳失败时应提示用户重新登录
func exampleHeartbeat(client *donghao.Client) {
	fmt.Println("【示例3】心跳检测")
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
		log.Printf("心跳失败: %v\n", err)
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

// exampleGetUser 演示获取用户信息
//
// 返回的用户信息包括:
//   - 用户名、到期时间、剩余积分等
func exampleGetUser(client *donghao.Client) {
	fmt.Println("【示例4】获取用户信息")
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
		log.Printf("获取用户信息失败: %v\n", err)
		fmt.Println()
		return
	}

	if result.IsSuccess() {
		fmt.Printf("用户信息获取成功!\n")
		fmt.Printf("返回数据: %v\n", result.Data)
	} else {
		fmt.Printf("获取失败: %s\n", result.Msg())
	}

	fmt.Println()
}

// exampleUserData 演示用户数据存取
//
// 用户数据说明:
//   - 每个用户可存储自定义数据
//   - 数据格式为字符串，可存储JSON序列化后的数据
//   - 数据会持久化保存，下次登录仍可获取
func exampleUserData(client *donghao.Client) {
	fmt.Println("【示例5】用户数据管理")
	fmt.Println("----------------------------------------")

	fmt.Println("5.1 存储用户数据")
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
		log.Printf("存储用户数据失败: %v\n", err)
	} else if result.IsSuccess() {
		fmt.Printf("存储成功: %v\n", result.Data)
	} else {
		fmt.Printf("存储失败: %s\n", result.Msg())
	}

	fmt.Println("\n5.2 获取用户数据")
	userData, err := client.GetUdata(
		"testuser",
		"token123",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("获取用户数据失败: %v\n", err)
	} else if userData.IsSuccess() {
		fmt.Printf("用户数据: %v\n", userData.Data)
	} else {
		fmt.Printf("获取失败: %s\n", userData.Msg())
	}

	fmt.Println()
}

// exampleVariable 演示云变量管理
//
// 云变量特点:
//   - 用户级别的键值对存储
//   - 支持设置、获取、删除操作
func exampleVariable(client *donghao.Client) {
	fmt.Println("【示例6】变量管理")
	fmt.Println("----------------------------------------")

	fmt.Println("6.1 设置变量")
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
		log.Printf("设置变量失败: %v\n", err)
	} else if result.IsSuccess() {
		fmt.Printf("设置成功: %v\n", result.Data)
	} else {
		fmt.Printf("设置失败: %s\n", result.Msg())
	}

	fmt.Println("\n6.2 获取变量")
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
		log.Printf("获取变量失败: %v\n", err)
	} else if value.IsSuccess() {
		varValue, decodeErr := value.GetVariableValue()
		if decodeErr != nil {
			fmt.Printf("解码失败: %v\n", decodeErr)
			fmt.Printf("原始数据: %v\n", value.GetResultMap())
		} else {
			fmt.Printf("变量值: %s\n", varValue)
		}
	} else {
		fmt.Printf("获取失败: %s\n", value.Msg())
	}

	fmt.Println("\n6.3 删除变量")
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
		log.Printf("删除变量失败: %v\n", err)
	} else if result2.IsSuccess() {
		fmt.Printf("删除成功: %v\n", result2.Data)
	} else {
		fmt.Printf("删除失败: %s\n", result2.Msg())
	}

	fmt.Println()
}

// exampleConstant 演示获取云端常量
//
// 常量说明:
//   - 软件级别的只读配置
//   - 所有用户共享
func exampleConstant(client *donghao.Client) {
	fmt.Println("【示例7】获取常量")
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
		log.Printf("获取常量失败: %v\n", err)
	} else if result.IsSuccess() {
		fmt.Printf("常量值: %v\n", result.Data)
	} else {
		fmt.Printf("获取失败: %s\n", result.Msg())
	}

	fmt.Println()
}

// exampleBlacklist 演示黑名单管理
//
// 黑名单功能说明:
//   - 支持IP、MAC、用户名等多种类型
//   - 可用于限制特定用户或设备的访问
func exampleBlacklist(client *donghao.Client) {
	fmt.Println("【示例8】黑名单管理")
	fmt.Println("----------------------------------------")

	fmt.Println("8.1 查询黑名单")
	result, err := client.GetBlack(
		"ip",
		"192.168.1.100",
	)

	if err != nil {
		log.Printf("查询黑名单失败: %v\n", err)
	} else if result.IsSuccess() {
		fmt.Printf("查询结果: %v\n", result.Data)
	} else {
		fmt.Printf("查询失败: %s\n", result.Msg())
	}

	fmt.Println("\n8.2 设置黑名单")
	result2, err := client.SetBlack(
		"ip",
		"192.168.1.100",
		"测试拉黑",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("设置黑名单失败: %v\n", err)
	} else if result2.IsSuccess() {
		fmt.Printf("设置成功: %v\n", result2.Data)
	} else {
		fmt.Printf("设置失败: %s\n", result2.Msg())
	}

	fmt.Println()
}

// exampleOtherFeatures 演示其他功能
//
// 包含功能:
//   - 卡密充值
//   - 修改密码
//   - 换绑信息（修改用户名/机器码/IP/QQ）
//   - 绑定推荐人
//   - 扣除积分
//   - 添加日志
//   - 用户登出
func exampleOtherFeatures(client *donghao.Client) {
	fmt.Println("【示例9】其他功能")
	fmt.Println("----------------------------------------")

	fmt.Println("9.1 卡密充值")
	result, err := client.Recharge(
		"testuser",
		"CARD-123456",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("充值失败: %v\n", err)
	} else if result.IsSuccess() {
		fmt.Printf("充值成功: %v\n", result.Data)
	} else {
		fmt.Printf("充值失败: %s\n", result.Msg())
	}

	fmt.Println("\n9.2 修改密码")
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
		log.Printf("修改密码失败: %v\n", err)
	} else if result2.IsSuccess() {
		fmt.Printf("修改密码成功: %v\n", result2.Data)
	} else {
		fmt.Printf("修改密码失败: %s\n", result2.Msg())
	}

	fmt.Println("\n9.3 换绑信息（修改用户名/机器码/IP/QQ）")
	result3, err := client.Binding(
		"testuser",
		"password",
		"newusername",    // 新用户名（为空则不修改）
		"AA:BB:CC:DD:EE:FF", // 新机器码（为空则不修改）
		"192.168.1.200", // 新IP地址（为空则不修改）
		"123456789",     // 新QQ号（为空则不修改）
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("换绑失败: %v\n", err)
	} else if result3.IsSuccess() {
		fmt.Printf("换绑成功: %v\n", result3.Data)
	} else {
		fmt.Printf("换绑失败: %s\n", result3.Msg())
	}

	fmt.Println("\n9.4 绑定推荐人")
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
		log.Printf("绑定推荐人失败: %v\n", err)
	} else if result4.IsSuccess() {
		fmt.Printf("绑定成功: %v\n", result4.Data)
	} else {
		fmt.Printf("绑定失败: %s\n", result4.Msg())
	}

	fmt.Println("\n9.5 扣除积分")
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
		log.Printf("扣除积分失败: %v\n", err)
	} else if result5.IsSuccess() {
		fmt.Printf("扣除成功: %v\n", result5.Data)
	} else {
		fmt.Printf("扣除失败: %s\n", result5.Msg())
	}

	fmt.Println("\n9.6 添加日志")
	result6, err := client.AddLog(
		"testuser",
		"[测试日志] 这是一条测试日志内容",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("添加日志失败: %v\n", err)
	} else if result6.IsSuccess() {
		fmt.Printf("添加日志成功: %v\n", result6.Data)
	} else {
		fmt.Printf("添加日志失败: %s\n", result6.Msg())
	}

	fmt.Println("\n9.7 验证账号密码")
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
		log.Printf("验证失败: %v\n", err)
	} else if result7.IsSuccess() {
		fmt.Printf("验证成功: %v\n", result7.Data)
	} else {
		fmt.Printf("验证失败: %s\n", result7.Msg())
	}

	fmt.Println("\n9.8 用户登出")
	result8, err := client.Logout(
		"testuser",
		"token123",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("登出失败: %v\n", err)
	} else if result8.IsSuccess() {
		fmt.Printf("登出成功: %v\n", result8.Data)
	} else {
		fmt.Printf("登出失败: %s\n", result8.Msg())
	}

	fmt.Println()
}

// exampleCloudFunctions 演示云计算函数
//
// 云计算功能说明:
//   - Func: 需要登录的云计算函数
//   - Func2: 无需登录的云计算函数
//   - CallPHP: 需要登录的PHP函数调用
//   - CallPHP2: 无需登录的PHP函数调用
func exampleCloudFunctions(client *donghao.Client) {
	fmt.Println("【示例10】云计算函数")
	fmt.Println("----------------------------------------")

	fmt.Println("10.1 云计算1 (需登录)")
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
		log.Printf("云计算1失败: %v\n", err)
	} else if result.IsSuccess() {
		fmt.Printf("云计算1成功: %v\n", result.Data)
	} else {
		fmt.Printf("云计算1失败: %s\n", result.Msg())
	}

	fmt.Println("\n10.2 云计算2 (无需登录)")
	result2, err := client.Func2(
		"jia",
		"1,2",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("云计算2失败: %v\n", err)
	} else if result2.IsSuccess() {
		fmt.Printf("云计算2成功: %v\n", result2.Data)
	} else {
		fmt.Printf("云计算2失败: %s\n", result2.Msg())
	}

	fmt.Println("\n10.3 调用PHP (需登录)")
	result3, err := client.CallPHP(
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
		log.Printf("调用PHP失败: %v\n", err)
	} else if result3.IsSuccess() {
		fmt.Printf("调用PHP成功: %v\n", result3.Data)
	} else {
		fmt.Printf("调用PHP失败: %s\n", result3.Msg())
	}

	fmt.Println("\n10.4 调用PHP2 (无需登录)")
	result4, err := client.CallPHP2(
		"jia",
		"1,2",
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("调用PHP2失败: %v\n", err)
	} else if result4.IsSuccess() {
		fmt.Printf("调用PHP2成功: %v\n", result4.Data)
	} else {
		fmt.Printf("调用PHP2失败: %s\n", result4.Msg())
	}

	fmt.Println()
}

// exampleLoginCard 演示卡密登录
//
// 卡密登录说明:
//   - 使用卡密直接登录，无需用户名和密码
//   - 需要后端开启"充值卡登录模式"(dl_type=1)
//   - 卡密作为用户名，密码为空
//   - 如果卡密未使用，会自动创建用户
//
// 适用场景:
//   - 发卡平台销售卡密
//   - 用户直接使用卡密登录软件
func exampleLoginCard(client *donghao.Client) {
	fmt.Println("【示例1.5】卡密登录")
	fmt.Println("----------------------------------------")

	// 使用卡密登录
	// 注意: 这里使用示例卡密，实际使用时请替换为真实卡密
	card := "YKUvGvYSuVFFTp41T1WO"

	fmt.Printf("正在使用卡密登录: %s ...\n", card)

	result, err := client.LoginCard(
		card,
		"1.0.0",
		"00:11:22:33:44:55",
		"192.168.1.100",
		"client001",
	)

	if err != nil {
		log.Printf("卡密登录请求失败: %v\n", err)
		fmt.Println()
		return
	}

	if result.IsSuccess() {
		fmt.Println("✅ 卡密登录成功!")
		fmt.Printf("   Token: %s\n", client.GetToken())
		fmt.Printf("   用户名: %s\n", client.GetCurrentUser())

		// 显示用户数据
		data, err := result.GetData()
		if err == nil && data != "" {
			fmt.Printf("   用户数据: %s\n", data)
		}

		// 显示其他信息
		m := result.GetResultMap()
		if m != nil {
			if endtime, ok := m["endtime"].(string); ok {
				fmt.Printf("   到期时间: %s\n", endtime)
			}
			if points, ok := m["points"].(float64); ok {
				fmt.Printf("   剩余点数: %.0f\n", points)
			}
		}
	} else {
		fmt.Printf("❌ 卡密登录失败: %s\n", result.Msg())
		fmt.Println("   提示: 请确认后端已开启充值卡登录模式(dl_type=1)")
	}

	fmt.Println()
}

// exampleAutoHeartbeat 演示自动心跳 - 已禁用
//
// 自动心跳功能暂不可用，相关方法已被注释
//
// 自动心跳说明:
//   - 启动后台goroutine自动发送心跳包
//   - 维持在线状态
//   - 心跳间隔由SetHeartbeatInterval设置
//
// func exampleAutoHeartbeat(client *donghao.Client) {
// 	fmt.Println("【示例11】自动心跳")
// 	fmt.Println("----------------------------------------")
//
// 	client.SetHeartbeatInterval(60)
//
// 	fmt.Println("启动自动心跳...")
// 	client.StartAutoHeartbeat(
// 		"testuser",
// 		"token123",
// 		"1.0.0",
// 		"00:11:22:33:44:55",
// 		"192.168.1.100",
// 		"client001",
// 	)
//
// 	fmt.Println("自动心跳已启动，每60秒发送一次")
// 	fmt.Println("运行5秒后停止...")
//
// 	time.Sleep(5 * time.Second)
//
// 	client.StopAutoHeartbeat()
// 	fmt.Println("自动心跳已停止")
//
// 	fmt.Println()
// }
