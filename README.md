# Grok 聊天记录导出工具

一个用Go编写的 Grok 聊天记录导出工具
可以自动提取完整对话历史并保存为文本文件。

## 功能特点

- 自动处理 Grok 页面的懒加载机制，提取完整历史消息
- 支持 Cloudflare 验证自动绕过
- 使用 Cookie 认证，无需登录
- 自动根据对话标题命名文件
- 无头浏览器模式，后台运行

## 前置要求

1. **Chrome 浏览器**：程序需要调用 Chrome 或 Chromium 浏览器
   - Windows：[下载 Chrome](https://www.google.com/chrome/)
   - 如果已安装 Edge 浏览器也可以（基于 Chromium）
   - 程序会自动查找系统中的 Chrome/Chromium

2. **Go 环境**（如果选择方法一）：Go 1.21 或更高版本

## 安装

### 方法一：直接运行（需要 Go 环境）

```bash
git clone https://github.com/Xxin-Cpp/Grok-ChatExporter.git
cd Grok-ChatExporter
go run main.go  # 首次运行会自动下载依赖
```

### 方法二：编译成可执行文件

```bash
git clone https://github.com/Xxin-Cpp/Grok-ChatExporter.git
cd Grok-ChatExporter
go build -o Grok-ChatExporter.exe main.go  # 自动下载依赖并编译
```

编译后会生成 `Grok-ChatExporter.exe`，可以直接运行，不需要 Go 环境。

## 配置

### 1. 提取 Cookie

需要从浏览器中提取 Grok 的 Cookie：

**使用浏览器扩展（推荐）：**

1. 安装 Cookie 导出扩展：
   - Chrome/Edge：[Cookie-Editor](https://chromewebstore.google.com/detail/hlkenndednhfkekhgcdicdfddnkalmdm?utm_source=item-share-cb)

2. 登录 [grok.com](https://grok.com)

3. 点击扩展图标 → 导出 Cookie（JSON 格式）

4. 复制导出的JSON数据粘贴至config.yml

### 2. 配置文件

在项目目录创建 `config.yml` 文件：

```yaml
cookies: |
  [
    {"domain":".grok.com","name":"sso","value":"你的cookie值","path":"/","httpOnly":true,"secure":true,"sameSite":"lax"},
    {"domain":".grok.com","name":"sso-rw","value":"你的cookie值","path":"/","httpOnly":true,"secure":true,"sameSite":"lax"},
    {"domain":".grok.com","name":"x-userid","value":"你的cookie值","path":"/","httpOnly":false,"secure":true,"sameSite":"lax"},
    {"domain":".grok.com","name":"_ga","value":"你的cookie值","path":"/","httpOnly":false,"secure":false,"sameSite":"lax"},
    {"domain":".grok.com","name":"_ga_8FEWB057YH","value":"你的cookie值","path":"/","httpOnly":false,"secure":false,"sameSite":"lax"},
    {"domain":".grok.com","name":"i18nextLng","value":"zh-CN","path":"/","httpOnly":false,"secure":false,"sameSite":"lax"},
    {"domain":".grok.com","name":"mp_mixpanel__c","value":"你的cookie值","path":"/","httpOnly":false,"secure":false,"sameSite":"lax"}
  ]
```

**注意：** 将 `"你的cookie值"` 替换为实际的 Cookie 值。如果使用扩展导出，直接粘贴导出的 JSON 数组即可。

## 使用方法

### 1. 获取对话链接

1. 打开 Grok 网站
2. 进入要导出的对话
3. 复制浏览器地址栏的完整链接，格式类似：
   ```
   https://grok.com/c/xxxxxxxxx?rid=xxxxxxxxx
   ```

### 2. 运行程序

```bash
# 如果使用源码运行
go run main.go

# 如果使用编译后的可执行文件
./Grok-ChatExporter.exe
```

### 3. 输入链接

程序会提示：
```
输入Grok聊天链接:
```

粘贴刚才复制的链接，按回车。

### 4. 等待导出

程序会自动：
- 打开无头浏览器
- 处理 Cloudflare 验证（如果有）
- 滚动加载所有历史消息
- 提取对话内容
- 保存为文本文件

导出完成后会显示：
```
成功提取 x 条消息
聊天记录已成功导出到 对话标题.txt
```

## 输出格式

导出的文本文件格式：

```
用户: 第一条用户消息内容

xAi: Grok的回复内容

用户: 第二条用户消息内容

xAi: Grok的回复内容

...
```

## 常见问题

### Cookie 失效怎么办？

Cookie 有有效期，通常几个月后会失效。如果程序提示"未能提取到聊天记录"，需要重新登录 Grok 并提取新的 Cookie。

### 提取不完整怎么办？

程序会自动滚动加载历史消息。如果对话特别长（几百条），可能需要多等一会儿。程序会在连续 5 次滚动无新消息后自动停止。

### 遇到 Cloudflare 验证

程序内置了自动绕过逻辑，会尝试：
1. 检测验证页面
2. 自动点击验证框
3. 等待验证完成

通常 5-15 秒内会自动通过。

### 提示找不到 Chrome 浏览器

如果系统没有安装 Chrome，会报错。解决方法：
- 安装 [Google Chrome](https://www.google.com/chrome/)
- 或确保已安装 Edge 浏览器（Windows 10/11 自带）
- 程序会自动在常见路径查找浏览器

### 文件名乱码

如果对话标题包含特殊字符，程序会自动替换为下划线。如果标题获取失败，会使用默认文件名 `未命名对话.txt`。

## 技术说明

- 使用 chromedp 控制无头 Chrome 浏览器
- 通过 CSS 选择器识别消息气泡
- 自动处理懒加载和滚动
- 支持去重和角色识别

## 注意事项

- Cookie 属于敏感信息，请勿分享给他人
- 本工具仅供个人备份使用
- 请遵守 Grok 服务条款
- 导出的对话内容请妥善保管

## 许可证

本项目采用 [MIT License](LICENSE) 开源协议。
