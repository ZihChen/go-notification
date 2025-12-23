package errmsg

import "errors"

var (
	ErrRepoMerchantNotFound       = errors.New("repo merchant not found")
	ErrRepoDeleteMerchantNotFound = errors.New("repo delete merchant not found")
	ErrRepoPushKeyNotFound        = errors.New("repo push key not found")
	ErrRepoManagerNotFound        = errors.New("repo manager not found")
	ErrRepoDeleteManagerNotFound  = errors.New("repo delete manager not found")
	ErrRepoPlayerNotFound         = errors.New("repo player not found")
	ErrRepoDeletePlayerNotFound   = errors.New("repo delete player not found")
	ErrRepoLevelNotFound          = errors.New("repo level not found")
	ErrRepoTagNotFound            = errors.New("repo tag not found")
	ErrRepoAgentNotFound          = errors.New("repo agent not found")
	ErrUnknownEventType           = errors.New("unknown event type")

	// 位元遮罩相關錯誤
	ErrInvalidNotificationType = errors.New(
		"invalid notification type: value out of range or contains undefined bits",
	)
	ErrNotificationTypeRequired    = errors.New("notification type is required")
	ErrUnsupportedNotificationType = errors.New("unsupported notification type combination")
)
