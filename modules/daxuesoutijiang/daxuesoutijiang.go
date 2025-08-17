package daxuesoutijiang

import (
	"fmt"
	"log"
	"sync"
	"textsurf/modules"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

type DaxuesoutijiangModule struct {
	// 为每个模块实例添加一个互斥锁来保护页面访问
	pageMutex sync.Mutex
}

func NewDaxuesoutijiangModule() modules.Module {
	return &DaxuesoutijiangModule{}
}

func (m *DaxuesoutijiangModule) Name() string {
	return "daxuesoutijiang"
}

func (m *DaxuesoutijiangModule) GetLoginQRCode(session *modules.Session) (string, error) {
	// 访问大学生搜题匠首页
	log.Println("正在访问大学生搜题匠首页...")
	page := session.Browser.MustPage("https://www.daxuesoutijiang.com/")
	session.Page = page

	// 等待页面加载
	log.Println("等待页面加载...")
	page.MustWaitLoad()
	time.Sleep(2 * time.Second)

	// 点击登录按钮
	log.Println("点击登录按钮...")
	loginButton, err := page.Element("#main > div.header-container > header > div > div.header-nav > button")
	if err != nil {
		return "", fmt.Errorf("无法找到登录按钮: %v", err)
	}
	loginButton.MustClick()
	time.Sleep(2 * time.Second)

	// 等待二维码加载
	log.Println("等待二维码加载...")
	page.MustWaitLoad()
	time.Sleep(5 * time.Second) // 增加等待时间

	// 查找二维码canvas元素
	log.Println("查找二维码canvas元素...")
	qrElement, err := page.Element("#dx-login-dialog-container > div > div > div.login-by-qrcode-wrapper > div > div.login-by-qrcode-content > canvas")
	if err != nil {
		log.Println("无法找到二维码canvas元素: ", err)
		return "", fmt.Errorf("无法找到二维码canvas元素: %v", err)
	}

	log.Println("找到二维码canvas元素，尝试获取截图...")

	// 直接获取canvas元素的截图
	imgBytes, err := qrElement.Screenshot(proto.PageCaptureScreenshotFormatPng, 100)
	if err != nil {
		return "", fmt.Errorf("无法获取canvas截图: %v", err)
	}

	// 将图片保存到会话数据中
	session.Data["qrImage"] = imgBytes
	log.Println("成功获取二维码图片内容")

	// 返回一个标识符，表示图片已保存在会话中
	return "session_image", nil
}

// GetLoginQRCodeImage 获取登录二维码图片内容
func (m *DaxuesoutijiangModule) GetLoginQRCodeImage(session *modules.Session) ([]byte, error) {
	// 如果已经获取过二维码图片，直接返回
	if imgData, ok := session.Data["qrImage"].([]byte); ok {
		return imgData, nil
	}

	// 否则重新获取二维码
	_, err := m.GetLoginQRCode(session)
	if err != nil {
		return nil, err
	}

	// 再次尝试获取图片数据
	if imgData, ok := session.Data["qrImage"].([]byte); ok {
		return imgData, nil
	}

	return nil, fmt.Errorf("无法获取二维码图片")
}

func (m *DaxuesoutijiangModule) CheckLogin(session *modules.Session) (bool, map[string]string, error) {
	// 检查页面是否已初始化
	if session.Page == nil {
		return false, nil, fmt.Errorf("会话页面未初始化，请先获取登录二维码")
	}

	// 使用互斥锁保护对页面的访问，防止并发访问导致的竞态条件
	m.pageMutex.Lock()
	defer m.pageMutex.Unlock()

	// 检查是否已经登录成功
	// 登录成功后页面URL可能会变化，或者出现用户相关信息
	currentURL := session.Page.MustInfo().URL
	log.Printf("当前页面URL: %s", currentURL)

	// 检查是否有用户相关信息元素
	_, err := session.Page.Element(".user-info")
	if err == nil {
		// 获取cookies
		cookies, err := session.Page.Cookies([]string{})
		if err != nil {
			return false, nil, fmt.Errorf("获取cookies失败: %v", err)
		}

		// 转换为字符串map
		cookieMap := make(map[string]string)
		for _, cookie := range cookies {
			cookieMap[cookie.Name] = cookie.Value
		}

		return true, cookieMap, nil
	}

	// 检查是否有登录失败的提示
	_, err = session.Page.Element(".error-message")
	if err == nil {
		return false, nil, fmt.Errorf("登录失败，请重新尝试")
	}

	// 如果都没有明确结果，返回未登录状态
	return false, nil, nil
}

func (m *DaxuesoutijiangModule) Close(session *modules.Session) error {
	if session.Browser != nil {
		session.Browser.MustClose()
	}
	return nil
}
