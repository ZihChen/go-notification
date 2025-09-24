package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/utils/security"
)

// PushNotificationService App推播服務實作
type PushNotificationService struct {
	httpClient *http.Client
	baseURL    string
	logger     infrastructure.Logger
}

// NewPushNotificationService 創建App推播服務
func NewPushNotificationService(
	baseURL string,
	logger infrastructure.Logger,
) service.PushNotificationService {
	return &PushNotificationService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		logger:  logger,
	}
}

// SendPushNotification 發送App推播通知
func (s *PushNotificationService) SendPushNotification(
	ctx context.Context,
	apiKey string,
	req *service.PushNotificationRequest,
) (*service.PushNotificationResponse, error) {
	// 構建請求體
	requestBody, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorLog("Failed to marshal push notification request",
			s.logger.Error("error", err),
			s.logger.Any("request", req))
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 創建HTTP請求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.baseURL, bytes.NewBuffer(requestBody))
	if err != nil {
		s.logger.ErrorLog("Failed to create HTTP request",
			s.logger.Error("error", err),
			s.logger.String("url", s.baseURL))
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 設置請求頭
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)

	// 發送請求
	s.logger.InfoLog("Sending push notification",
		s.logger.String("url", s.baseURL),
		s.logger.Int("accounts_count", len(req.Accounts)),
		s.logger.String("title", req.MsgTitle))

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		s.logger.ErrorLog("Failed to send push notification request",
			s.logger.Error("error", err),
			s.logger.String("url", s.baseURL))
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// 讀取回應
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.ErrorLog("Failed to read response body",
			s.logger.Error("error", err),
			s.logger.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 檢查狀態碼
	if resp.StatusCode != http.StatusOK {
		s.logger.WarnLog("Push notification request failed",
			s.logger.Int("status_code", resp.StatusCode),
			s.logger.String("response_body", string(responseBody)),
			s.logger.String("api_key", security.SanitizeAPIKey(apiKey)))

		return &service.PushNotificationResponse{
			Success: false,
			Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(responseBody)),
		}, nil
	}

	s.logger.InfoLog("Push notification sent successfully",
		s.logger.Int("status_code", resp.StatusCode),
		s.logger.Int("accounts_count", len(req.Accounts)),
		s.logger.String("response_body", string(responseBody)))

	return &service.PushNotificationResponse{
		Success: true,
		Message: string(responseBody),
	}, nil
}
