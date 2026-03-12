package modules

import (
	"time"

	"github.com/go-rod/rod"
)

// Session 会话信息
type Session struct {
	ID        string
	Browser   *rod.Browser
	Page      *rod.Page
	CreatedAt time.Time
	Module    Module
	Data      map[string]interface{} // 存储模块特定数据
}

// Module 接口定义
type Module interface {
	// Name 返回模块名称
	Name() string

	// GetLoginQRCode 获取登录二维码URL
	// 返回二维码图片的URL
	GetLoginQRCode(session *Session) (string, error)

	// GetLoginQRCodeImage 获取登录二维码图片内容
	// 返回二维码图片的字节数据
	GetLoginQRCodeImage(session *Session) ([]byte, error)

	// PrepareSMSLogin 准备短信登录页面
	// 返回登录页面相关信息
	PrepareSMSLogin(session *Session) (info map[string]interface{}, err error)

	// SendSMSCode 发送短信验证码
	// phoneNumber: 手机号码
	SendSMSCode(session *Session, phoneNumber string) error

	// VerifySMSCode 验证短信验证码并提交登录
	// smsCode: 短信验证码
	VerifySMSCode(session *Session, smsCode string) error

	// CheckLogin 检查是否登录成功
	// 返回是否登录成功和错误信息
	CheckLogin(session *Session) (bool, map[string]string, error)

	// Close 关闭会话资源
	Close(session *Session) error
}

// ModuleRegistry 模块注册表
type ModuleRegistry struct {
	modules map[string]Module
}

// NewModuleRegistry 创建新的模块注册表
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules: make(map[string]Module),
	}
}

// Register 注册模块
func (r *ModuleRegistry) Register(module Module) {
	r.modules[module.Name()] = module
}

// Get 获取模块
func (r *ModuleRegistry) Get(name string) (Module, bool) {
	module, exists := r.modules[name]
	return module, exists
}

// List 获取所有模块名称
func (r *ModuleRegistry) List() []string {
	names := make([]string, 0, len(r.modules))
	for name := range r.modules {
		names = append(names, name)
	}
	return names
}
