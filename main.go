package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/urfave/cli/v2"
)

var browser *rod.Browser

// 配置结构体
type Config struct {
	Port     string
	Headless bool
	Debug    bool
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

// 启动服务器
func startServer(config Config) error {
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
	fmt.Printf("GET http://localhost%s/fetch/text?url=https://example.com\n", config.Port)
	fmt.Printf("GET http://localhost%s/fetch/html?url=https://example.com&css_path=.content\n", config.Port)
	fmt.Printf("GET http://localhost%s/fetch/text?url=https://example.com&click_css_path=.load-more&css_path=.result\n", config.Port)
	fmt.Printf("\nService URL: http://localhost%s\n", config.Port)

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
