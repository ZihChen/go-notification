package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/handler"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/logger"
	"github.com/robfig/cron/v3"
)

// ExampleJob 示例排程任務
type ExampleJob struct {
	name string
}

// Execute 實現 ScheduledJob 介面
func (j *ExampleJob) Execute(ctx context.Context) error {
	fmt.Printf("[%s] 執行示例任務: %s\n", time.Now().Format("2006-01-02 15:04:05"), j.name)
	// 模擬任務執行時間
	time.Sleep(2 * time.Second)
	fmt.Printf("[%s] 任務完成: %s\n", time.Now().Format("2006-01-02 15:04:05"), j.name)
	return nil
}

// GetName 實現 ScheduledJob 介面
func (j *ExampleJob) GetName() string {
	return j.name
}

func main() {
	// 創建配置
	cfg := &config.Config{
		App: config.AppConfig{
			Name:  "scheduler-example",
			Env:   "local",
			Debug: true,
		},
	}

	// 創建日誌
	serviceLogger := logger.NewServiceLogger(cfg)

	// 創建 Scheduler Handler
	schedulerHandler := handler.NewSchedulerHandler(serviceLogger)

	// 註冊不同類型的任務

	// 1. 每5秒執行一次的任務
	err := schedulerHandler.RegisterJobWithInterval(
		"every-5-seconds",
		&ExampleJob{name: "5秒任務"},
		5*time.Second,
	)
	if err != nil {
		log.Fatalf("註冊5秒任務失敗: %v", err)
	}

	// 2. 每30秒執行一次的任務
	err = schedulerHandler.RegisterJobWithInterval(
		"every-30-seconds",
		&ExampleJob{name: "30秒任務"},
		30*time.Second,
	)
	if err != nil {
		log.Fatalf("註冊30秒任務失敗: %v", err)
	}

	// 3. 使用Cron表達式：每分鐘執行一次
	err = schedulerHandler.RegisterJobWithSchedule(
		"every-minute",
		&ExampleJob{name: "每分鐘任務"},
		"0 * * * * *",
	)
	if err != nil {
		log.Fatalf("註冊每分鐘任務失敗: %v", err)
	}

	// 4. 使用Cron表達式：每10秒執行一次
	err = schedulerHandler.RegisterJobWithSchedule(
		"every-10-seconds-cron",
		&ExampleJob{name: "Cron 10秒任務"},
		"*/10 * * * * *",
	)
	if err != nil {
		log.Fatalf("註冊Cron 10秒任務失敗: %v", err)
	}

	// 創建 Cron 調度器
	cronManager := cron.New(cron.WithSeconds())

	// 註冊所有任務到調度器
	schedulerHandler.RegisterJobs(cronManager)

	// 啟動調度器
	cronManager.Start()
	fmt.Println("排程器已啟動，已註冊的任務:")
	for i, jobName := range schedulerHandler.GetRegisteredJobs() {
		fmt.Printf("  %d. %s\n", i+1, jobName)
	}

	// 運行2分鐘後停止
	fmt.Println("任務將運行2分鐘...")
	time.Sleep(2 * time.Minute)

	// 停止調度器
	cronCtx := cronManager.Stop()
	<-cronCtx.Done()
	fmt.Println("排程器已停止")
}
