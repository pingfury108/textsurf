package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"textsurf/modules"
	"textsurf/modules/baidu"
	"textsurf/sessions"

	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/urfave/cli/v2"
)

// 全局变量存储配置
var (
	browser        *rod.Browser
	moduleRegistry *modules.ModuleRegistry
	sessionManager *sessions.Manager
	config         Config // 添加这行来存储全局配置
)

// 配置结构体
type Config struct {
	Port     string
	Headless bool
	Debug    bool
}

// 初始化模块注册表
func initModuleRegistry() {
	moduleRegistry = modules.NewModuleRegistry()

	// 注册百度模块
	baiduModule := baidu.NewBaiduModule()
	moduleRegistry.Register(baiduModule)

	fmt.Println("Module registry initialized")
}

// 初始化会话管理器
func initSessionManager() {
	sessionManager = sessions.NewManager()
	fmt.Println("Session manager initialized")
}

// 初始化浏览器实例
func initBrowser(headless bool) {
	url := launcher.New().
		Headless(headless).
		MustLaunch()

	browser = rod.New().ControlURL(url).MustConnect()
	fmt.Printf("Browser initialized successfully (headless: %v)\n", headless)
}

// 清理浏览器资源
func closeBrowser() {
	if browser != nil {
		browser.MustClose()
		fmt.Println("Browser closed")
	}
}

// API 处理函数
func handleRequest(c *gin.Context) {
	// 获取返回类型（text 或 html）
	returnType := c.Param("type")
	if returnType != "text" && returnType != "html" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid return type. Use 'text' or 'html'",
		})
		return
	}

	// 获取必需的 URL 参数
	targetURL := c.Query("url")
	if targetURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required parameter: url",
		})
		return
	}

	// 获取可选参数
	cssPath := c.Query("css_path")
	clickCssPath := c.Query("click_css_path")

	// 创建新页面
	page := browser.MustPage(targetURL)
	defer page.MustClose()

	// 等待页面加载完成
	page.MustWaitStable()
	time.Sleep(1 * time.Second) // 额外等待确保页面完全加载

	// 如果提供了点击路径，先执行点击操作
	if clickCssPath != "" {
		fmt.Printf("Attempting to click element with CSS path: %s\n", clickCssPath)
		clickElement, err := page.Element(clickCssPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Error finding click element with CSS path '%s': %v", clickCssPath, err),
			})
			return
		}

		err = clickElement.Click(proto.InputMouseButtonLeft, 1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Error clicking element: %v", err),
			})
			return
		}

		// 等待点击后的内容加载
		time.Sleep(2 * time.Second)
		page.MustWaitStable()
		fmt.Println("Successfully clicked element")
	}

	var content string
	var err error

	// 根据是否提供 CSS 路径来获取内容
	if cssPath != "" {
		// 获取指定 CSS 路径的内容
		fmt.Printf("Getting content from CSS path: %s\n", cssPath)
		element, err := page.Element(cssPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Error finding element with CSS path '%s': %v", cssPath, err),
			})
			return
		}

		if returnType == "text" {
			content, err = element.Text()
		} else {
			content, err = element.HTML()
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Error getting %s content from element: %v", returnType, err),
			})
			return
		}
	} else {
		// 获取整个页面的内容
		fmt.Println("Getting full page content")
		bodyElement := page.MustElement("body")

		if returnType == "text" {
			content, err = bodyElement.Text()
		} else {
			content, err = bodyElement.HTML()
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Error getting page %s content: %v", returnType, err),
			})
			return
		}
	}

	// 返回结果
	response := gin.H{
		"url":            targetURL,
		"type":           returnType,
		"content":        content,
		"css_path":       cssPath,
		"click_css_path": clickCssPath,
	}

	c.JSON(http.StatusOK, response)
}

// 创建会话
func handleCreateSession(c *gin.Context) {
	moduleName := c.Param("module")

	// 获取模块
	module, exists := moduleRegistry.Get(moduleName)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Module '%s' not found", moduleName),
		})
		return
	}

	// 创建会话
	session, err := sessionManager.CreateSession(module, config.Headless) // 使用全局配置的 headless 设置
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create session: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id": session.ID,
		"module":     moduleName,
		"created_at": session.CreatedAt,
	})
}

// 获取登录二维码
func handleGetLoginQRCode(c *gin.Context) {
	sessionID := c.Param("session_id")
	moduleName := c.Param("module")

	log.Printf("收到获取二维码请求: module=%s, session_id=%s\n", moduleName, sessionID)

	// 获取会话
	session, exists := sessionManager.GetSession(sessionID)
	if !exists {
		log.Printf("会话不存在: %s\n", sessionID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Session '%s' not found", sessionID),
		})
		return
	}

	// 验证模块
	if session.Module.Name() != moduleName {
		log.Printf("模块不匹配: session.module=%s, requested_module=%s\n", session.Module.Name(), moduleName)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Session does not belong to module '%s'", moduleName),
		})
		return
	}

	log.Printf("调用模块 %s 的 GetLoginQRCodeImage 方法\n", moduleName)
	// 获取二维码图片内容
	qrCodeImage, err := session.Module.GetLoginQRCodeImage(session)
	if err != nil {
		log.Printf("获取二维码失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get QR code: %v", err),
		})
		return
	}

	// 返回图片内容
	c.Data(http.StatusOK, "image/png", qrCodeImage)
}

// 检查登录状态
func handleCheckLogin(c *gin.Context) {
	sessionID := c.Param("session_id")
	moduleName := c.Param("module")

	// 获取会话
	session, exists := sessionManager.GetSession(sessionID)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Session '%s' not found", sessionID),
		})
		return
	}

	// 验证模块
	if session.Module.Name() != moduleName {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Session does not belong to module '%s'", moduleName),
		})
		return
	}

	// 检查登录状态
	loggedIn, cookies, err := session.Module.CheckLogin(session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to check login status: %v", err),
		})
		return
	}

	if loggedIn {
		// 登录成功，转换cookies为分号分隔的字符串
		cookieStr := ""
		for name, value := range cookies {
			if cookieStr != "" {
				cookieStr += "; "
			}
			cookieStr += name + "=" + value
		}

		// 返回cookies并关闭会话
		c.JSON(http.StatusOK, gin.H{
			"cookies": cookieStr,
		})

		// 关闭会话
		sessionManager.DeleteSession(sessionID)
	} else {
		// 仍在等待登录
		c.JSON(http.StatusOK, gin.H{
			"session_id": sessionID,
			"module":     moduleName,
			"logged_in":  false,
			"message":    "Waiting for user to scan QR code and login",
		})
	}
}

// 启动服务器
func startServer(cfg Config) error {
	// 存储全局配置
	config = cfg

	// 初始化模块注册表和会话管理器
	initModuleRegistry()
	initSessionManager()

	// 初始化浏览器
	initBrowser(config.Headless)
	defer closeBrowser()

	// 设置 Gin 模式
	if !config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建 Gin 路由器
	r := gin.Default()

	// 设置路由 - path 参数指定返回类型（text 或 html）
	r.GET("/fetch/:type", handleRequest)

	// 新增模块化登录相关路由
	// 创建会话
	r.POST("/api/:module/session", handleCreateSession)

	// 获取登录二维码
	r.GET("/api/:module/:session_id/login_img", handleGetLoginQRCode)

	// 检查登录状态并获取cookies
	r.GET("/api/:module/:session_id/cookies", handleCheckLogin)

	// 添加健康检查接口
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"browser":  "connected",
			"headless": config.Headless,
			"port":     config.Port,
		})
	})

	// 添加使用说明接口
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Rod Browser Web Service",
			"version": "1.0.0",
			"usage": map[string]interface{}{
				"endpoint":        "/fetch/{type}",
				"types":           []string{"text", "html"},
				"required_params": []string{"url"},
				"optional_params": []string{"css_path", "click_css_path"},
				"examples": []string{
					"/fetch/text?url=https://example.com",
					"/fetch/html?url=https://example.com&css_path=.content",
					"/fetch/text?url=https://example.com&click_css_path=.load-more&css_path=.result",
				},
			},
			"modules": moduleRegistry.List(),
			"config": map[string]interface{}{
				"port":     config.Port,
				"headless": config.Headless,
				"debug":    config.Debug,
			},
		})
	})

	// 启动服务器
	fmt.Printf("Starting Rod Browser Web Service on port %s...\n", config.Port)
	fmt.Printf("Headless mode: %v\n", config.Headless)
	fmt.Printf("Debug mode: %v\n", config.Debug)
	fmt.Println("\nAPI Examples:")
	fmt.Printf("GET http://localhost:%s/fetch/text?url=https://example.com\n", config.Port)
	fmt.Printf("GET http://localhost:%s/fetch/html?url=https://example.com&css_path=.content\n", config.Port)
	fmt.Printf("GET http://localhost:%s/fetch/text?url=https://example.com&click_css_path=.load-more&css_path=.result\n", config.Port)
	fmt.Printf("POST http://localhost:%s/api/baidu/session\n", config.Port)
	fmt.Printf("GET http://localhost:%s/api/baidu/{session_id}/login_img\n", config.Port)
	fmt.Printf("GET http://localhost:%s/api/baidu/{session_id}/cookies\n", config.Port)
	fmt.Printf("\nService URL: http://localhost:%s\n", config.Port)

	return r.Run(":" + config.Port)
}

func main() {
	app := &cli.App{
		Name:  "textsurf",
		Usage: "Rod Browser Web Service - 基于 Rod 浏览器的网页内容提取服务",
		Description: `TextSurf 是一个基于 Rod 浏览器的 web 服务，提供网页内容提取功能。
支持获取整个页面或特定元素的文本/HTML内容，
还支持点击操作后再提取内容。`,
		Version: "1.0.0",
		Authors: []*cli.Author{
			{
				Name:  "TextSurf",
				Email: "support@textsurf.com",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   ":8080",
				Usage:   "服务器监听端口",
				EnvVars: []string{"TEXTSURF_PORT"},
			},
			&cli.BoolFlag{
				Name:    "headless",
				Aliases: []string{"H"},
				Value:   false,
				Usage:   "启用无头浏览器模式",
				EnvVars: []string{"TEXTSURF_HEADLESS"},
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Value:   false,
				Usage:   "启用调试模式",
				EnvVars: []string{"TEXTSURF_DEBUG"},
			},
		},
		Action: func(ctx *cli.Context) error {
			config := Config{
				Port:     ctx.String("port"),
				Headless: ctx.Bool("headless"),
				Debug:    ctx.Bool("debug"),
			}

			return startServer(config)
		},
		Commands: []*cli.Command{
			{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "显示版本信息",
				Action: func(ctx *cli.Context) error {
					fmt.Printf("TextSurf v%s\n", ctx.App.Version)
					fmt.Println("基于 Rod 浏览器的网页内容提取服务")
					return nil
				},
			},
			{
				Name:  "test",
				Usage: "测试浏览器连接",
				Action: func(ctx *cli.Context) error {
					fmt.Println("Testing browser connection...")
					initBrowser(true) // 使用无头模式测试
					defer closeBrowser()

					page := browser.MustPage("https://httpbin.org/get")
					defer page.MustClose()
					page.MustWaitStable()

					fmt.Println("✅ Browser connection test successful!")
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
