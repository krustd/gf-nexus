package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gclient"
)

const (
	adminAddr = "http://localhost:8081"
)

func main() {
	ctx := context.Background()
	client := g.Client()

	// 示例：创建命名空间
	fmt.Println("\n=== 1. 创建命名空间 ===")
	createNamespace(ctx, client, "myapp", "我的应用", "示例应用的配置命名空间")

	// 示例：保存草稿配置
	fmt.Println("\n=== 2. 保存草稿配置 ===")
	draftConfig := `
server:
  port: 8080
  host: "0.0.0.0"

database:
  dsn: "mysql://localhost:3306/myapp"

features:
  new_ui: true
  beta_feature: false
`
	saveDraft(ctx, client, "myapp", "app.yaml", strings.TrimSpace(draftConfig), "yaml")

	// 示例：发布配置
	fmt.Println("\n=== 3. 发布配置 ===")
	publishConfig(ctx, client, "myapp", "app.yaml")

	// 示例：配置灰度规则（30% 的客户端使用新版本）
	fmt.Println("\n=== 4. 设置灰度规则（30%）===")
	saveGrayRule(ctx, client, "myapp", "app.yaml", 30, true)

	// 示例：更新草稿（新版本）
	fmt.Println("\n=== 5. 更新草稿（灰度版本）===")
	draftConfigV2 := `
server:
  port: 8080
  host: "0.0.0.0"

database:
  dsn: "mysql://localhost:3306/myapp_v2"

features:
  new_ui: true
  beta_feature: true  # 新特性开启
`
	saveDraft(ctx, client, "myapp", "app.yaml", strings.TrimSpace(draftConfigV2), "yaml")

	// 示例：查询配置列表
	fmt.Println("\n=== 6. 查询配置列表 ===")
	listConfigs(ctx, client, "myapp")

	fmt.Println("\n=== 完成 ===")
	fmt.Println("现在可以启动 client 示例来测试长轮询和配置变更通知")
}

func createNamespace(ctx context.Context, client *gclient.Client, id, name, desc string) {
	resp, err := client.Post(ctx, adminAddr+"/api/v1/namespaces/", map[string]interface{}{
		"id":          id,
		"name":        name,
		"description": desc,
	})
	if err != nil {
		fmt.Printf("创建命名空间失败: %v\n", err)
		return
	}
	defer resp.Close()
	printResponse(resp)
}

func saveDraft(ctx context.Context, client *gclient.Client, namespace, key, value, format string) {
	resp, err := client.Post(ctx, adminAddr+"/api/v1/configs/draft", map[string]interface{}{
		"namespace": namespace,
		"key":       key,
		"value":     value,
		"format":    format,
	})
	if err != nil {
		fmt.Printf("保存草稿失败: %v\n", err)
		return
	}
	defer resp.Close()
	printResponse(resp)
}

func publishConfig(ctx context.Context, client *gclient.Client, namespace, key string) {
	resp, err := client.Post(ctx, adminAddr+"/api/v1/configs/publish", map[string]interface{}{
		"namespace": namespace,
		"key":       key,
	})
	if err != nil {
		fmt.Printf("发布配置失败: %v\n", err)
		return
	}
	defer resp.Close()
	printResponse(resp)
}

func saveGrayRule(ctx context.Context, client *gclient.Client, namespace, key string, percentage int, enabled bool) {
	resp, err := client.Post(ctx, adminAddr+"/api/v1/gray/", map[string]interface{}{
		"namespace":  namespace,
		"key":        key,
		"percentage": percentage,
		"enabled":    enabled,
	})
	if err != nil {
		fmt.Printf("设置灰度规则失败: %v\n", err)
		return
	}
	defer resp.Close()
	printResponse(resp)
}

func listConfigs(ctx context.Context, client *gclient.Client, namespace string) {
	resp, err := client.Get(ctx, adminAddr+"/api/v1/configs/list?namespace="+namespace)
	if err != nil {
		fmt.Printf("查询配置列表失败: %v\n", err)
		return
	}
	defer resp.Close()
	printResponse(resp)
}

func printResponse(resp *gclient.Response) {
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("请求失败: HTTP %d\n", resp.StatusCode)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.ReadAll(), &result); err != nil {
		fmt.Printf("解析响应失败: %v\n", err)
		return
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}
