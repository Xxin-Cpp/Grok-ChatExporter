package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Cookies string `yaml:"cookies"`
}

type Cookie struct {
	Domain         string  `json:"domain"`
	ExpirationDate float64 `json:"expirationDate,omitempty"`
	HostOnly       bool    `json:"hostOnly"`
	HTTPOnly       bool    `json:"httpOnly"`
	Name           string  `json:"name"`
	Path           string  `json:"path"`
	SameSite       string  `json:"sameSite"`
	Secure         bool    `json:"secure"`
	Session        bool    `json:"session"`
	Value          string  `json:"value"`
}

type Message struct {
	Role    string
	Content string
}

func main() {
	// 先关掉日志
	log.SetOutput(io.Discard)

	configData, err := os.ReadFile("config.yml")
	if err != nil {
		fmt.Printf("读取config.yml失败: %v\n", err)
		fmt.Println("请确保config.yml文件存在于当前目录")
		os.Exit(1)
	}

	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		fmt.Printf("解析config.yml失败: %v\n", err)
		os.Exit(1)
	}

	var cookies []Cookie
	if err := json.Unmarshal([]byte(config.Cookies), &cookies); err != nil {
		fmt.Printf("解析cookies失败: %v\n", err)
		fmt.Println("请检查config.yml中的cookies格式是否正确")
		os.Exit(1)
	}

	fmt.Println("输入Grok聊天链接:")
	reader := bufio.NewReader(os.Stdin)
	chatURL, _ := reader.ReadString('\n')
	chatURL = strings.TrimSpace(chatURL)

	if chatURL == "" {
		fmt.Println("链接不能为空")
		os.Exit(1)
	}

	// 开爬
	messages, title, err := scrapeChatHistory(chatURL, cookies)
	if err != nil {
		fmt.Printf("爬取失败: %v\n", err)
		os.Exit(1)
	}

	filename := sanitizeFilename(title) + ".txt"
	if filename == ".txt" || filename == "_.txt" {
		filename = "chat_history.txt"
	}

	if err := exportToTxt(messages, filename); err != nil {
		fmt.Printf("导出失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("聊天记录已成功导出到 %s\n", filename)
}

// 主要
func scrapeChatHistory(url string, cookies []Cookie) ([]Message, string, error) {

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("exclude-switches", "enable-automation"),
		chromedp.Flag("disable-extensions", false),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"),
		chromedp.WindowSize(1920, 1080),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	var messages []Message
	var title string

	headers := map[string]interface{}{
		"sec-ch-ua":                   `"Google Chrome";v="143", "Chromium";v="143", "Not A(Brand";v="24"`,
		"sec-ch-ua-arch":              `"x86"`,
		"sec-ch-ua-bitness":           `"64"`,
		"sec-ch-ua-full-version":      `"143.0.7499.193"`,
		"sec-ch-ua-full-version-list": `"Google Chrome";v="143.0.7499.193", "Chromium";v="143.0.7499.193", "Not A(Brand";v="24.0.0.0"`,
		"sec-ch-ua-mobile":            `?0`,
		"sec-ch-ua-model":             `""`,
		"sec-ch-ua-platform":          `"Windows"`,
		"sec-ch-ua-platform-version":  `"15.0.0"`,
		"sec-fetch-dest":              "empty",
		"sec-fetch-mode":              "cors",
		"sec-fetch-site":              "same-origin",
	}

	err := chromedp.Run(ctx,
		network.Enable(),
		network.SetExtraHTTPHeaders(headers),

		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `
				Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
				Object.defineProperty(navigator, 'plugins', {get: () => [1, 2, 3, 4, 5]});
				Object.defineProperty(navigator, 'languages', {get: () => ['zh-CN', 'zh', 'en']});
				window.chrome = {runtime: {}};
			`
			var res interface{}
			return chromedp.Evaluate(script, &res).Do(ctx)
		}),

		// 先打开主页
		chromedp.Navigate("https://grok.com"),
		chromedp.Sleep(2*time.Second),

		// ck
		chromedp.ActionFunc(func(ctx context.Context) error {
			for _, cookie := range cookies {
				expiry := cdp.TimeSinceEpoch(time.Now().Add(365 * 24 * time.Hour))
				if cookie.ExpirationDate > 0 {
					expiry = cdp.TimeSinceEpoch(time.Unix(int64(cookie.ExpirationDate), 0))
				}

				sameSite := network.CookieSameSiteLax
				if cookie.SameSite == "strict" {
					sameSite = network.CookieSameSiteStrict
				} else if cookie.SameSite == "none" {
					sameSite = network.CookieSameSiteNone
				}

				err := network.SetCookie(cookie.Name, cookie.Value).
					WithDomain(cookie.Domain).
					WithPath(cookie.Path).
					WithHTTPOnly(cookie.HTTPOnly).
					WithSecure(cookie.Secure).
					WithSameSite(sameSite).
					WithExpires(&expiry).
					Do(ctx)

				if err != nil {
					continue
				}
			}
			return nil
		}),

		// 跳转到目标页面
		chromedp.Navigate(url),
		chromedp.Sleep(10*time.Second),

		// 傻逼cf
		chromedp.ActionFunc(func(ctx context.Context) error {
			var cfChallenge bool
			checkCF := `
				(function() {
					const indicators = [
						document.title.includes('Just a moment'),
						document.title.includes('Attention Required'),
						document.querySelector('#challenge-form'),
						document.querySelector('.cf-browser-verification'),
						document.querySelector('[name="cf-turnstile-response"]'),
						document.querySelector('iframe[src*="challenges.cloudflare.com"]')
					];
					return indicators.some(x => x);
				})();
			`
			chromedp.Evaluate(checkCF, &cfChallenge).Do(ctx)

			if cfChallenge {
				// 尝试点击验证框
				clickScript := `
					(function() {
						const iframe = document.querySelector('iframe[src*="challenges.cloudflare.com"]');
						if (iframe) {
							iframe.click();
							return true;
						}

						const buttons = [
							document.querySelector('input[type="button"][value*="Verify"]'),
							document.querySelector('button[type="submit"]'),
							document.querySelector('.cf-turnstile'),
							document.querySelector('#challenge-stage button')
						];
						
						for (let btn of buttons) {
							if (btn) {
								btn.click();
								return true;
							}
						}
						
						return false;
					})();
				`
				var clicked bool
				chromedp.Evaluate(clickScript, &clicked).Do(ctx)

				// 让cf完成验证
				if !clicked {
					time.Sleep(8 * time.Second)
				} else {
					time.Sleep(5 * time.Second)
				}

				// 检测
				var stillChallenged bool
				chromedp.Evaluate(checkCF, &stillChallenged).Do(ctx)

				if stillChallenged {
					time.Sleep(10 * time.Second)
				}
			}

			return nil
		}),

		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),

		// 滚动加载历史，别删！！！！！！
		chromedp.ActionFunc(func(ctx context.Context) error {
			maxAttempts := 50
			noChangeCount := 0

			for i := 0; i < maxAttempts; i++ {
				var beforeCount int
				countScript := `document.querySelectorAll('.message-bubble, [class*="message-bubble"]').length;`
				chromedp.Evaluate(countScript, &beforeCount).Do(ctx)

				scrollScript := `
					(function() {
						const scrollContainer = document.querySelector('.overflow-y-auto') || 
						                       document.querySelector('[class*="overflow-y"]') ||
						                       document.querySelector('main [class*="overflow"]');
						
						if (scrollContainer) {
							scrollContainer.scrollTop = 0;
							scrollContainer.dispatchEvent(new Event('scroll'));

							setTimeout(() => {
								scrollContainer.scrollTop = 50;
								scrollContainer.dispatchEvent(new Event('scroll'));
								setTimeout(() => {
									scrollContainer.scrollTop = 0;
									scrollContainer.dispatchEvent(new Event('scroll'));
								}, 100);
							}, 100);
							
							return true;
						} else {
							window.scrollTo(0, 0);
							return false;
						}
					})();
				`
				chromedp.Evaluate(scrollScript, nil).Do(ctx)

				time.Sleep(5 * time.Second)

				var afterCount int
				chromedp.Evaluate(countScript, &afterCount).Do(ctx)

				if afterCount == beforeCount {
					noChangeCount++
					if noChangeCount >= 5 {
						break
					}
				} else {
					noChangeCount = 0
				}
			}

			return nil
		}),

		// 滚到底，确保
		chromedp.Evaluate(`
			const scrollContainer = document.querySelector('.overflow-y-auto') || document.body;
			scrollContainer.scrollTop = scrollContainer.scrollHeight;
		`, nil),
		chromedp.Sleep(3*time.Second),

		chromedp.ActionFunc(func(ctx context.Context) error {
			time.Sleep(2 * time.Second)

			var titleText string
			titleScript := `
				(function() {
					const titleSelectors = [
						'main h1',
						'[role="main"] h1',
						'header h1',
						'h1[class*="text"]',
						'div[class*="conversation"] h1',
						'div[class*="chat"] h1',
						'h1'
					];
					
					for (let selector of titleSelectors) {
						const elem = document.querySelector(selector);
						if (elem && elem.textContent.trim()) {
							const text = elem.textContent.trim();
							if (text !== 'Grok' && 
							    text !== 'Home' && 
							    text !== 'Menu' && 
							    text.length > 3 &&
							    !/^\d+$/.test(text)) {
								return text;
							}
						}
					}
					
					const firstUserMsg = document.querySelector('.rounded-bl-lg');
					if (firstUserMsg) {
						const text = firstUserMsg.textContent.trim();
						return text.length > 30 ? text.substring(0, 30) + '...' : text;
					}
					
					return '未命名对话';
				})();
			`
			chromedp.Evaluate(titleScript, &titleText).Do(ctx)
			title = titleText

			// 提取所有消息
			var result string
			script := `
				(function() {
					let messages = [];
					const messageBubbles = document.querySelectorAll('.message-bubble, [class*="message-bubble"]');
					
					for (let bubble of messageBubbles) {
						const text = (bubble.textContent || '').trim();
						if (text.length < 5) continue;
						
						const className = bubble.className || '';
						let role = 'unknown';

						if (className.includes('rounded-br-lg')) {
							role = 'assistant';
						} else if (className.includes('rounded-bl-lg')) {
							role = 'user';
						} else {
							let parent = bubble.parentElement;
							while (parent && parent !== document.body) {
								const parentClass = parent.className || '';
								if (parentClass.includes('items-end') || parentClass.includes('justify-end')) {
									role = 'assistant';
									break;
								} else if (parentClass.includes('items-start') || parentClass.includes('justify-start')) {
									role = 'user';
									break;
								}
								parent = parent.parentElement;
							}
						}
						
						if (role === 'unknown') {
							role = text.length > 500 ? 'assistant' : 'user';
						}
						
						messages.push({
							role: role,
							content: text
						});
					}
					
					const dedupedMessages = [];
					for (let i = 0; i < messages.length; i++) {
						if (i === 0 || messages[i].content !== messages[i-1].content) {
							dedupedMessages.push(messages[i]);
						}
					}
					
					return JSON.stringify(dedupedMessages);
				})();
			`
			if err := chromedp.Evaluate(script, &result).Do(ctx); err != nil {
				return err
			}

			var extractedMessages []map[string]interface{}
			if err := json.Unmarshal([]byte(result), &extractedMessages); err != nil {
				return fmt.Errorf("无法解析消息: %v", err)
			}

			fmt.Printf("成功提取 %d 条消息\n", len(extractedMessages))

			for _, msg := range extractedMessages {
				role := "xAi"
				if r, ok := msg["role"].(string); ok && r == "assistant" {
					role = "用户"
				}
				content := ""
				if c, ok := msg["content"].(string); ok {
					content = c
				}

				if content != "" && !strings.Contains(content, "chromedp") && !strings.Contains(content, "func ") {
					messages = append(messages, Message{
						Role:    role,
						Content: content,
					})
				}
			}

			return nil
		}),
	)

	if err != nil {
		return nil, "", fmt.Errorf("浏览器自动化执行失败: %v", err)
	}

	if len(messages) == 0 {
		return nil, "", fmt.Errorf("未能提取到聊天记录，请检查链接和cookie是否有效")
	}

	return messages, title, nil
}

// 导出
func exportToTxt(messages []Message, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, msg := range messages {
		_, err := file.WriteString(fmt.Sprintf("%s: %s\n\n", msg.Role, msg.Content))
		if err != nil {
			return err
		}
	}

	return nil
}

func sanitizeFilename(name string) string {
	illegal := regexp.MustCompile(`[<>:"/\\|?*]`)
	name = illegal.ReplaceAllString(name, "_")

	if len(name) > 100 {
		name = name[:100]
	}

	return strings.TrimSpace(name)
}
