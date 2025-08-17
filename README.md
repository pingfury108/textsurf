# TextSurf - 网页内容提取服务

TextSurf 是一个基于 [Rod](https://github.com/go-rod/rod) 浏览器的 web 服务，提供网页内容提取功能。支持获取整个页面或特定元素的文本/HTML内容，还支持点击操作后再提取内容。

## 功能特性

1. 提取网页文本或HTML内容
2. 支持CSS选择器定位特定元素
3. 支持点击操作后再提取内容
4. 模块化设计，支持多种网站登录流程
5. 会话隔离，每个用户请求独立处理

## 安装和运行

```bash
# 克隆项目
git clone <repository-url>
cd textsurf

# 下载依赖
go mod tidy

# 运行服务
go run main.go

# 或者编译后运行
go build -o textsurf
./textsurf
```

## 基本用法

### 提取网页内容

```bash
# 提取整个页面的文本内容
curl "http://localhost:8080/fetch/text?url=https://example.com"

# 提取特定元素的HTML内容
curl "http://localhost:8080/fetch/html?url=https://example.com&css_path=.content"

# 点击按钮后再提取内容
curl "http://localhost:8080/fetch/text?url=https://example.com&click_css_path=.load-more&css_path=.result"
```

## 模块化登录功能

TextSurf 支持模块化登录功能，可以为不同网站实现登录流程。

### 百度登录示例

1. 创建会话：
```bash
curl -X POST http://localhost:8080/api/baidu/session
```
响应：
```json
{
  "session_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "module": "baidu",
  "created_at": "2023-01-01T00:00:00Z"
}
```

2. 获取登录二维码：
```bash
curl http://localhost:8080/api/baidu/{session_id}/login_img
```
响应：
```json
{
  "session_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "module": "baidu",
  "qr_code_url": "https://passport.baidu.com/qr/image?xxx"
}
```

3. 检查登录状态并获取cookies：
```bash
curl http://localhost:8080/api/baidu/{session_id}/cookies
```

如果用户尚未扫码登录：
```json
{
  "session_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "module": "baidu",
  "logged_in": false,
  "message": "Waiting for user to scan QR code and login"
}
```

如果用户已扫码登录：
```json
{
  "session_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "module": "baidu",
  "logged_in": true,
  "cookies": {
    "BDUSS": "xxxxxxxxxx",
    "STOKEN": "xxxxxxxxxx",
    // ... 其他cookies
  }
}
```

## 配置选项

```
--port, -p     服务器监听端口 (默认: :8080)
--headless, -H 启用无头浏览器模式 (默认: false)
--debug, -d    启用调试模式 (默认: false)
```

环境变量：
- `TEXTSURF_PORT` - 服务器监听端口
- `TEXTSURF_HEADLESS` - 启用无头浏览器模式
- `TEXTSURF_DEBUG` - 启用调试模式

## 开发

### 添加新模块

1. 在 `modules/` 目录下创建新模块目录
2. 实现 `modules.Module` 接口
3. 在 `main.go` 的 `initModuleRegistry` 函数中注册模块

### Module 接口

```go
type Module interface {
    // Name 返回模块名称
    Name() string
    
    // GetLoginQRCode 获取登录二维码
    // 返回二维码图片的URL或base64编码
    GetLoginQRCode(session *Session) (string, error)
    
    // CheckLogin 检查是否登录成功
    // 返回是否登录成功和错误信息
    CheckLogin(session *Session) (bool, map[string]string, error)
    
    // Close 关闭会话资源
    Close(session *Session) error
}
```

## API 接口说明

### 根路径 `/`
- 方法: GET
- 说明: 获取服务信息和使用说明

### 健康检查 `/health`
- 方法: GET
- 说明: 检查服务健康状态

### 内容提取 `/fetch/{type}`
- 方法: GET
- 参数:
  - `type`: 返回类型 (text 或 html)
  - `url`: 目标网址 (必需)
  - `css_path`: CSS选择器 (可选)
  - `click_css_path`: 点击元素的CSS选择器 (可选)

### 创建会话 `/api/{module}/session`
- 方法: POST
- 说明: 为指定模块创建新的登录会话

### 获取二维码 `/api/{module}/{session_id}/login_img`
- 方法: GET
- 说明: 获取指定会话的登录二维码

### 检查登录状态 `/api/{module}/{session_id}/cookies`
- 方法: GET
- 说明: 检查登录状态，登录成功后返回cookies

## 许可证

MIT