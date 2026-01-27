package metrics

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Metrics OpenTelemetry Metrics 服務
type Metrics struct {
	// API Metrics
	apiRequests metric.Int64Counter     // API 請求計數
	apiDuration metric.Float64Histogram // API 延遲分佈
	apiErrors   metric.Int64Counter     // API 錯誤計數

	// Campaign Metrics
	campaignProcessed metric.Int64Counter     // Campaign 處理計數
	campaignDuration  metric.Float64Histogram // Campaign 處理延遲
	campaignErrors    metric.Int64Counter     // Campaign 處理錯誤

	// Message Metrics
	messageSent      metric.Int64Counter // 訊息發送計數
	messageDelivered metric.Int64Counter // 訊息送達計數
	messageFailed    metric.Int64Counter // 訊息失敗計數

	// Consumer Metrics
	eventReceived  metric.Int64Counter     // KDS 事件接收計數
	eventProcessed metric.Int64Counter     // 事件處理計數
	eventDuration  metric.Float64Histogram // 事件處理延遲

	// Worker Metrics
	taskProcessed metric.Int64Counter     // 任務處理計數
	taskDuration  metric.Float64Histogram // 任務處理延遲

	// Infrastructure
	meterProvider *sdkmetric.MeterProvider
	logger        infrastructure.Logger
	enabled       bool
}

// NewMetrics 建立新的 Metrics 服務
func NewMetrics(cfg *config.Config, log infrastructure.Logger) (*Metrics, error) {
	m := &Metrics{
		logger:  log,
		enabled: cfg.Metrics.Enabled && cfg.Metrics.OTLP.Enabled,
	}

	if !m.enabled {
		log.InfoLog("metrics is disabled")
		return m, nil
	}

	// 建立 OTLP HTTP Exporter
	exporter, err := m.createExporter(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// 建立 Resource
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.App.Name),
		semconv.ServiceVersion("1.0.0"),
		semconv.DeploymentEnvironment(cfg.App.Env),
	)

	// 設置導出間隔
	interval := cfg.Metrics.OTLP.Interval
	if interval == 0 {
		interval = 60 * time.Second
	}

	// 建立 MeterProvider
	m.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(interval)),
		),
	)

	// 設置全域 MeterProvider
	otel.SetMeterProvider(m.meterProvider)

	// 建立 Meter
	meter := m.meterProvider.Meter("fat-notification-cat")

	// 初始化所有 metrics
	if err = m.initMetrics(meter); err != nil {
		return nil, fmt.Errorf("failed to init metrics: %w", err)
	}

	log.InfoLog("metrics service initialized successfully",
		log.String("endpoint", cfg.Metrics.OTLP.Endpoint),
		log.String("path", cfg.Metrics.OTLP.Path),
		log.String("interval", interval.String()))

	return m, nil
}

// createExporter 建立 OTLP HTTP Exporter
func (m *Metrics) createExporter(cfg *config.Config) (sdkmetric.Exporter, error) {
	// 處理 endpoint，移除協議前綴
	endpoint := cfg.Metrics.OTLP.Endpoint
	isHTTPS := true
	if strings.HasPrefix(endpoint, "https://") {
		endpoint = strings.TrimPrefix(endpoint, "https://")
	} else if strings.HasPrefix(endpoint, "http://") {
		endpoint = strings.TrimPrefix(endpoint, "http://")
		isHTTPS = false
	}

	// 建立自定義 HTTP Client，強制使用 HTTP/1.1 避免 HTTP/2 GOAWAY 問題
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			ForceAttemptHTTP2:   false,
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
		Timeout: cfg.Metrics.OTLP.Timeout,
	}

	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithHTTPClient(httpClient),
	}

	// 設置 URL Path
	if cfg.Metrics.OTLP.Path != "" {
		opts = append(opts, otlpmetrichttp.WithURLPath(cfg.Metrics.OTLP.Path))
	}

	// 設置是否使用不安全連線
	if cfg.Metrics.OTLP.Insecure || !isHTTPS {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	// 設置 Headers
	if len(cfg.Metrics.OTLP.Headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(cfg.Metrics.OTLP.Headers))
	}

	// 設置 Timeout
	if cfg.Metrics.OTLP.Timeout > 0 {
		opts = append(opts, otlpmetrichttp.WithTimeout(cfg.Metrics.OTLP.Timeout))
	}

	return otlpmetrichttp.New(context.Background(), opts...)
}

// initMetrics 初始化所有 metrics
func (m *Metrics) initMetrics(meter metric.Meter) error {
	var err error

	// === API Metrics ===
	m.apiRequests, err = meter.Int64Counter(
		"notification_api_requests_total",
		metric.WithDescription("Total number of API requests"),
	)
	if err != nil {
		return err
	}

	m.apiDuration, err = meter.Float64Histogram(
		"notification_api_duration_seconds",
		metric.WithDescription("API request duration in seconds"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5),
	)
	if err != nil {
		return err
	}

	m.apiErrors, err = meter.Int64Counter(
		"notification_api_errors_total",
		metric.WithDescription("Total number of API errors"),
	)
	if err != nil {
		return err
	}

	// === Campaign Metrics ===
	m.campaignProcessed, err = meter.Int64Counter(
		"notification_campaign_processed_total",
		metric.WithDescription("Total number of campaigns processed"),
	)
	if err != nil {
		return err
	}

	m.campaignDuration, err = meter.Float64Histogram(
		"notification_campaign_duration_seconds",
		metric.WithDescription("Campaign processing duration in seconds"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30),
	)
	if err != nil {
		return err
	}

	m.campaignErrors, err = meter.Int64Counter(
		"notification_campaign_errors_total",
		metric.WithDescription("Total number of campaign processing errors"),
	)
	if err != nil {
		return err
	}

	// === Message Metrics ===
	m.messageSent, err = meter.Int64Counter(
		"notification_message_sent_total",
		metric.WithDescription("Total number of messages sent"),
	)
	if err != nil {
		return err
	}

	m.messageDelivered, err = meter.Int64Counter(
		"notification_message_delivered_total",
		metric.WithDescription("Total number of messages delivered"),
	)
	if err != nil {
		return err
	}

	m.messageFailed, err = meter.Int64Counter(
		"notification_message_failed_total",
		metric.WithDescription("Total number of messages failed"),
	)
	if err != nil {
		return err
	}

	// === Consumer Metrics ===
	m.eventReceived, err = meter.Int64Counter(
		"notification_event_received_total",
		metric.WithDescription("Total number of KDS events received"),
	)
	if err != nil {
		return err
	}

	m.eventProcessed, err = meter.Int64Counter(
		"notification_event_processed_total",
		metric.WithDescription("Total number of events processed"),
	)
	if err != nil {
		return err
	}

	m.eventDuration, err = meter.Float64Histogram(
		"notification_event_duration_seconds",
		metric.WithDescription("Event processing duration in seconds"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1),
	)
	if err != nil {
		return err
	}

	// === Worker Metrics ===
	m.taskProcessed, err = meter.Int64Counter(
		"notification_task_processed_total",
		metric.WithDescription("Total number of worker tasks processed"),
	)
	if err != nil {
		return err
	}

	m.taskDuration, err = meter.Float64Histogram(
		"notification_task_duration_seconds",
		metric.WithDescription("Worker task processing duration in seconds"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.5, 1, 5, 10),
	)
	if err != nil {
		return err
	}

	return nil
}

// === API Metrics Methods ===

// RecordAPIRequest 記錄 API 請求
func (m *Metrics) RecordAPIRequest(method, path, status string) {
	if !m.enabled || m.apiRequests == nil {
		return
	}
	m.apiRequests.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("path", path),
			attribute.String("status", status),
		),
	)
}

// RecordAPIDuration 記錄 API 延遲
func (m *Metrics) RecordAPIDuration(method, path string, duration float64) {
	if !m.enabled || m.apiDuration == nil {
		return
	}
	m.apiDuration.Record(context.Background(), duration,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("path", path),
		),
	)
}

// RecordAPIError 記錄 API 錯誤
func (m *Metrics) RecordAPIError(method, path, errorType string) {
	if !m.enabled || m.apiErrors == nil {
		return
	}
	m.apiErrors.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("path", path),
			attribute.String("error_type", errorType),
		),
	)
}

// === Campaign Metrics Methods ===

// RecordCampaignProcessed 記錄 Campaign 處理
func (m *Metrics) RecordCampaignProcessed(merchantID string, campaignType string) {
	if !m.enabled || m.campaignProcessed == nil {
		return
	}
	m.campaignProcessed.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("merchant_id", merchantID),
			attribute.String("campaign_type", campaignType),
		),
	)
}

// RecordCampaignDuration 記錄 Campaign 處理延遲
func (m *Metrics) RecordCampaignDuration(merchantID string, duration float64) {
	if !m.enabled || m.campaignDuration == nil {
		return
	}
	m.campaignDuration.Record(context.Background(), duration,
		metric.WithAttributes(
			attribute.String("merchant_id", merchantID),
		),
	)
}

// RecordCampaignError 記錄 Campaign 處理錯誤
func (m *Metrics) RecordCampaignError(merchantID string, errorType string) {
	if !m.enabled || m.campaignErrors == nil {
		return
	}
	m.campaignErrors.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("merchant_id", merchantID),
			attribute.String("error_type", errorType),
		),
	)
}

// === Message Metrics Methods ===

// RecordMessageSent 記錄訊息發送
func (m *Metrics) RecordMessageSent(merchantID string, notificationType string) {
	if !m.enabled || m.messageSent == nil {
		return
	}
	m.messageSent.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("merchant_id", merchantID),
			attribute.String("notification_type", notificationType),
		),
	)
}

// RecordMessageDelivered 記錄訊息送達
func (m *Metrics) RecordMessageDelivered(merchantID string) {
	if !m.enabled || m.messageDelivered == nil {
		return
	}
	m.messageDelivered.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("merchant_id", merchantID),
		),
	)
}

// RecordMessageFailed 記錄訊息失敗
func (m *Metrics) RecordMessageFailed(merchantID string, reason string) {
	if !m.enabled || m.messageFailed == nil {
		return
	}
	m.messageFailed.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("merchant_id", merchantID),
			attribute.String("reason", reason),
		),
	)
}

// === Event Metrics Methods ===

// RecordEventReceived 記錄事件接收
func (m *Metrics) RecordEventReceived(eventType string) {
	if !m.enabled || m.eventReceived == nil {
		return
	}
	m.eventReceived.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("event_type", eventType),
		),
	)
}

// RecordEventProcessed 記錄事件處理完成
func (m *Metrics) RecordEventProcessed(eventType string, success bool) {
	if !m.enabled || m.eventProcessed == nil {
		return
	}
	m.eventProcessed.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("event_type", eventType),
			attribute.Bool("success", success),
		),
	)
}

// RecordEventDuration 記錄事件處理延遲
func (m *Metrics) RecordEventDuration(eventType string, duration float64) {
	if !m.enabled || m.eventDuration == nil {
		return
	}
	m.eventDuration.Record(context.Background(), duration,
		metric.WithAttributes(
			attribute.String("event_type", eventType),
		),
	)
}

// === Worker Metrics Methods ===

// RecordTaskProcessed 記錄任務處理
func (m *Metrics) RecordTaskProcessed(taskType string, success bool) {
	if !m.enabled || m.taskProcessed == nil {
		return
	}
	m.taskProcessed.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("task_type", taskType),
			attribute.Bool("success", success),
		),
	)
}

// RecordTaskDuration 記錄任務處理延遲
func (m *Metrics) RecordTaskDuration(taskType string, duration float64) {
	if !m.enabled || m.taskDuration == nil {
		return
	}
	m.taskDuration.Record(context.Background(), duration,
		metric.WithAttributes(
			attribute.String("task_type", taskType),
		),
	)
}

// === Lifecycle Methods ===

// Shutdown 關閉 Metrics 服務
func (m *Metrics) Shutdown(ctx context.Context) error {
	if m.meterProvider == nil {
		return nil
	}
	return m.meterProvider.Shutdown(ctx)
}

// IsEnabled 檢查 Metrics 是否啟用
func (m *Metrics) IsEnabled() bool {
	return m.enabled
}
