package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIO 连接配置（根据你的 config.yaml 修改）
const (
	endpoint        = "43.138.113.105:19000"
	accessKeyID     = "minio"
	secretAccessKey = "pmZMGPzY4ANyB6nn"
	useSSL          = false
)

// 所有需要删除的桶
var allBuckets = []string{
	"article-images",
	"community-assets",
	"community-resources",
	"document-images",
	"resource-chunks",
	"resource-previews",
	"system-assets",
	"temp-files",
	"user-avatars",
}

func main() {
	fmt.Println("🗑️  MinIO 清理工具 - 删除所有桶和数据")
	fmt.Println("==================================================")
	fmt.Printf("⚠️  警告：此操作将永久删除以下桶及其所有数据：\n")
	for _, bucket := range allBuckets {
		fmt.Printf("   - %s\n", bucket)
	}
	fmt.Println("==================================================")
	fmt.Println()

	// 等待3秒让用户看到警告
	fmt.Println("⏳ 3秒后开始执行...")
	time.Sleep(3 * time.Second)

	// 初始化 MinIO 客户端
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalf("❌ 连接 MinIO 失败: %v", err)
	}

	ctx := context.Background()

	// 删除每个桶
	totalDeleted := 0
	totalFailed := 0

	for _, bucketName := range allBuckets {
		fmt.Printf("\n🔄 处理桶: %s\n", bucketName)

		// 检查桶是否存在
		exists, err := client.BucketExists(ctx, bucketName)
		if err != nil {
			fmt.Printf("   ❌ 检查桶失败: %v\n", err)
			totalFailed++
			continue
		}

		if !exists {
			fmt.Printf("   ⏭️  桶不存在，跳过\n")
			continue
		}

		// 删除桶中所有对象
		objectCount := 0
		objectsCh := client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
			Recursive: true,
		})

		for object := range objectsCh {
			if object.Err != nil {
				fmt.Printf("   ❌ 列举对象失败: %v\n", object.Err)
				continue
			}

			// 删除对象
			err := client.RemoveObject(ctx, bucketName, object.Key, minio.RemoveObjectOptions{})
			if err != nil {
				fmt.Printf("   ❌ 删除对象失败 %s: %v\n", object.Key, err)
			} else {
				objectCount++
				if objectCount%100 == 0 {
					fmt.Printf("   🗑️  已删除 %d 个对象...\n", objectCount)
				}
			}
		}

		if objectCount > 0 {
			fmt.Printf("   ✅ 删除了 %d 个对象\n", objectCount)
		}

		// 删除空桶
		err = client.RemoveBucket(ctx, bucketName)
		if err != nil {
			fmt.Printf("   ❌ 删除桶失败: %v\n", err)
			totalFailed++
		} else {
			fmt.Printf("   ✅ 桶已删除\n")
			totalDeleted++
		}
	}

	// 统计结果
	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("📊 清理完成")
	fmt.Printf("✅ 成功删除: %d 个桶\n", totalDeleted)
	if totalFailed > 0 {
		fmt.Printf("❌ 删除失败: %d 个桶\n", totalFailed)
	}
	fmt.Println("==================================================")
}
