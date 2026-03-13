package baichuanweb

import (
	"fmt"
	"log"
	"sync"
	"textsurf/modules"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type BaichuanwebModule struct {
	pageMutex sync.Mutex
}

func NewBaichuanwebModule() modules.Module {
	return &BaichuanwebModule{}
}

func (m *BaichuanwebModule) Name() string {
	return "baichuanweb"
}

func (m *BaichuanwebModule) GetLoginQRCode(session *modules.Session) (string, error) {
	return "", fmt.Errorf("百川网页面不支持二维码登录")
}

func (m *BaichuanwebModule) GetLoginQRCodeImage(session *modules.Session) ([]byte, error) {
	return nil, fmt.Errorf("百川网页面不支持二维码登录")
}

func (m *BaichuanwebModule) PrepareSMSLogin(session *modules.Session) (map[string]interface{}, error) {
	m.pageMutex.Lock()
	defer m.pageMutex.Unlock()

	log.Println("正在访问百川网登录页面...")

	// 使用现有的 stealth 页面
	page := session.Page
	if page == nil {
		return nil, fmt.Errorf("会话页面未初始化")
	}

	// 导航到登录页面
	page.MustNavigate("https://www.baichuanweb.com/portal/login")

	log.Println("等待页面加载...")
	page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	info := map[string]interface{}{
		"status":     "ready",
		"login_type": "sms",
		"url":        page.MustInfo().URL,
	}

	log.Println("短信登录页面准备完成")
	return info, nil
}

func (m *BaichuanwebModule) SendSMSCode(session *modules.Session, phoneNumber string) error {
	m.pageMutex.Lock()
	defer m.pageMutex.Unlock()

	if session.Page == nil {
		return fmt.Errorf("会话页面未初始化，请先准备登录页面")
	}

	log.Printf("准备发送验证码到手机号: %s", phoneNumber)

	page := session.Page

	log.Println("1. 勾选用户协议复选框...")
	checkboxLabel, err := page.Element("label.el-checkbox")
	if err != nil {
		return fmt.Errorf("无法找到用户协议复选框: %v", err)
	}

	checkboxLabel.MustClick()
	time.Sleep(500 * time.Millisecond)
	log.Println("已点击用户协议复选框")

	log.Println("2. 输入手机号...")
	phoneInput, err := page.Element("input[maxlength='11'][placeholder*='手机号']")
	if err != nil {
		phoneInput, err = page.Element("input[maxlength='11'][placeholder*='输入']")
		if err != nil {
			return fmt.Errorf("无法找到手机号输入框: %v", err)
		}
	}

	phoneInput.MustInput("")
	phoneInput.MustInput(phoneNumber)
	time.Sleep(500 * time.Millisecond)
	log.Printf("手机号已输入: %s", phoneNumber)

	log.Println("3. 点击获取验证码按钮...")
	sendButton, err := page.Element(".code-right")
	if err != nil {
		return fmt.Errorf("无法找到获取验证码按钮: %v", err)
	}

	sendButton.MustClick()
	log.Println("已点击获取验证码按钮")

	time.Sleep(2 * time.Second)

	log.Println("验证码发送成功")
	return nil
}

func (m *BaichuanwebModule) VerifySMSCode(session *modules.Session, smsCode string) error {
	m.pageMutex.Lock()
	defer m.pageMutex.Unlock()

	if session.Page == nil {
		return fmt.Errorf("会话页面未初始化，请先准备登录页面")
	}

	log.Printf("准备验证短信验证码: %s", smsCode)

	page := session.Page

	log.Println("1. 输入验证码...")
	codeInput, err := page.Element("input[maxlength='6'][placeholder*='验证码']")
	if err != nil {
		return fmt.Errorf("无法找到验证码输入框: %v", err)
	}

	codeInput.MustInput("")
	codeInput.MustInput(smsCode)
	time.Sleep(500 * time.Millisecond)
	log.Printf("验证码已输入: %s", smsCode)

	log.Println("2. 点击登录按钮...")
	loginButton, err := page.Element(".login-btn")
	if err != nil {
		return fmt.Errorf("无法找到登录按钮: %v", err)
	}

	loginButton.MustClick()
	log.Println("已点击登录按钮")

	time.Sleep(3 * time.Second)

	log.Println("验证码验证完成，等待登录结果")
	return nil
}

func (m *BaichuanwebModule) CheckLogin(session *modules.Session) (bool, map[string]string, error) {
	if session.Page == nil {
		return false, nil, fmt.Errorf("会话页面未初始化，请先准备登录页面")
	}

	m.pageMutex.Lock()
	defer m.pageMutex.Unlock()

	if err := rod.Try(func() {
		session.Page.MustInfo()
	}); err != nil {
		return false, nil, fmt.Errorf("浏览器连接已关闭: %v", err)
	}

	currentURL := session.Page.MustInfo().URL
	log.Printf("当前页面URL: %s", currentURL)

	loginSuccess := false

	if currentURL != "https://www.baichuanweb.com/portal/login" {
		log.Println("URL已跳转，可能登录成功")
		loginSuccess = true
	}

	log.Println("检查登录成功标志...")
	_, err := session.Page.Element(".red-color")
	if err == nil {
		log.Println("找到退出登录按钮，登录成功")
		loginSuccess = true
	}

	_, err = session.Page.Element(".el-button--primary")
	if err == nil {
		log.Println("找到我的工作台按钮，登录成功")
		loginSuccess = true
	}

	if loginSuccess {
		// 检查是否已经缓存了cookies
		log.Printf("检查缓存，session.Data[cookies] 类型: %T, 值: %v", session.Data["cookies"], session.Data["cookies"])
		if cachedCookies, ok := session.Data["cookies"].(map[string]string); ok && len(cachedCookies) > 0 {
			log.Println("使用缓存的cookies")
			return true, cachedCookies, nil
		}

		log.Println("关闭弹出的横幅...")
		closeButtons, err := session.Page.Elements(".upload-banner-modal img.close")
		if err == nil {
			for _, btn := range closeButtons {
				if btn != nil {
					rod.Try(func() {
						btn.MustClick()
						log.Println("已关闭一个横幅")
					})
					time.Sleep(300 * time.Millisecond)
				}
			}
		}

		log.Println("正在获取cookies...")

		// 使用带超时的方式获取cookies
		var cookies []*proto.NetworkCookie
		var cookieErr error

		done := make(chan struct{})
		go func() {
			cookies, cookieErr = session.Page.Cookies([]string{"https://www.baichuanweb.com"})
			close(done)
		}()

		select {
		case <-done:
			if cookieErr != nil {
				log.Printf("获取cookies失败: %v", cookieErr)
				return false, nil, fmt.Errorf("获取cookies失败: %v", cookieErr)
			}
		case <-time.After(10 * time.Second):
			log.Println("获取cookies超时")
			return false, nil, fmt.Errorf("获取cookies超时")
		}

		cookieMap := make(map[string]string)
		for _, cookie := range cookies {
			cookieMap[cookie.Name] = cookie.Value
			log.Printf("  Cookie: %s = %s", cookie.Name, cookie.Value)
		}

		// 缓存cookies到session中，支持多次获取
		session.Data["cookies"] = cookieMap
		log.Printf("成功获取并缓存 %d 个cookies", len(cookieMap))
		return true, cookieMap, nil
	}

	_, err = session.Page.Element(".error-message")
	if err == nil {
		return false, nil, fmt.Errorf("登录失败，请检查验证码是否正确")
	}

	_, err = session.Page.Element(".login-failed")
	if err == nil {
		return false, nil, fmt.Errorf("登录失败，请检查验证码是否正确")
	}

	log.Println("登录状态未确定，返回未登录")
	return false, nil, nil
}

func (m *BaichuanwebModule) Close(session *modules.Session) error {
	if session.Browser != nil {
		session.Browser.MustClose()
	}
	return nil
}
