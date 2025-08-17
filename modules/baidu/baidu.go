package baidu

import (
	"fmt"
	"log"
	"sync"
	"textsurf/modules"
	"time"
)

type BaiduModule struct {
	// 为每个模块实例添加一个互斥锁来保护页面访问
	pageMutex sync.Mutex
}

func NewBaiduModule() modules.Module {
	return &BaiduModule{}
}

func (m *BaiduModule) Name() string {
	return "baidu"
}

func (m *BaiduModule) GetLoginQRCode(session *modules.Session) (string, error) {
	// 访问百度登录页面
	log.Println("正在访问百度登录页面...")
	page := session.Browser.MustPage("https://passport.baidu.com/v2/?login")
	session.Page = page

	// 等待页面加载
	log.Println("等待页面加载...")
	page.MustWaitLoad()
	time.Sleep(2 * time.Second)

	// 尝试切换到二维码登录 (如果默认不是二维码登录)
	// 使用 TryElement 来避免阻塞
	log.Println("检查是否有二维码登录选项...")
	if loginTypeSwitch, err := page.Element("a[data-type='qrcode']"); err == nil {
		log.Println("找到二维码登录选项，点击切换...")
		loginTypeSwitch.MustClick()
		time.Sleep(1 * time.Second)
	}

	// 等待二维码加载
	log.Println("等待二维码加载...")
	page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	// 查找二维码图片元素，使用你提供的精确选择器
	log.Println("查找二维码图片元素...")
	qrElement, err := page.Element("#TANGRAM__PSP_3__QrcodeMain > img")
	if err != nil {
		log.Println("无法找到二维码元素: ", err)
		return "", fmt.Errorf("无法找到二维码元素: %v", err)
	}

	log.Println("找到二维码元素，获取src属性...")
	// 获取二维码图片的 src
	qrSrc, err := qrElement.Attribute("src")
	if err != nil {
		return "", fmt.Errorf("无法获取二维码图片src: %v", err)
	}

	// 如果是相对路径，补全为绝对路径
	qrURL := *qrSrc
	if qrURL[0] == '/' {
		qrURL = "https://passport.baidu.com" + qrURL
	}

	// 保存二维码URL到会话数据中供后续检查使用
	session.Data["qrURL"] = qrURL

	log.Printf("成功获取二维码URL: %s\n", qrURL)
	return qrURL, nil
}

// GetLoginQRCodeImage 获取登录二维码图片内容
func (m *BaiduModule) GetLoginQRCodeImage(session *modules.Session) ([]byte, error) {
	// 访问百度登录页面
	log.Println("正在访问百度登录页面...")
	page := session.Browser.MustPage("https://passport.baidu.com/v2/?login")
	session.Page = page

	// 等待页面加载
	log.Println("等待页面加载...")
	page.MustWaitLoad()

	// 查找二维码图片元素，使用你提供的精确选择器
	log.Println("查找二维码图片元素...")
	qrElement, err := page.Element("#TANGRAM__PSP_3__QrcodeMain > img")
	if err != nil {
		log.Println("无法找到二维码元素: ", err)
		return nil, fmt.Errorf("无法找到二维码元素: %v", err)
	}

	log.Println("找到二维码元素，获取图片内容...")
	// 获取二维码图片内容
	imgBytes, err := qrElement.Resource()
	if err != nil {
		return nil, fmt.Errorf("无法获取二维码图片内容: %v", err)
	}

	log.Println("成功获取二维码图片内容")
	return imgBytes, nil
}

func (m *BaiduModule) CheckLogin(session *modules.Session) (bool, map[string]string, error) {
	// 检查页面是否已初始化
	if session.Page == nil {
		return false, nil, fmt.Errorf("会话页面未初始化，请先获取登录二维码")
	}

	// 使用互斥锁保护对页面的访问，防止并发访问导致的竞态条件
	m.pageMutex.Lock()
	defer m.pageMutex.Unlock()

	// 检查页面是否跳转到登录成功后的页面
	// 百度登录成功后通常会跳转到用户中心或首页
	url := session.Page.MustInfo().URL

	// 如果URL包含passport.baidu.com但不是登录页，则可能是登录成功了
	if url != "https://passport.baidu.com/v2/?login" &&
		(len(url) > 25 ||
			url == "https://www.baidu.com/" ||
			url == "https://passport.baidu.com/") {
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

	// 检查是否有登录成功的标志元素
	_, err := session.Page.Element(".user-name")
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

	// 尝试获取用户头像作为登录成功的标志
	_, err = session.Page.Element(".user-avatar")
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
	_, err = session.Page.Element(".pass-state-error")
	if err == nil {
		return false, nil, fmt.Errorf("登录失败，请重新尝试")
	}

	// 如果都没有明确结果，返回未登录状态
	return false, nil, nil
}

func (m *BaiduModule) Close(session *modules.Session) error {
	if session.Browser != nil {
		session.Browser.MustClose()
	}
	return nil
}
