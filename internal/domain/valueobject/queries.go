package valueobject

// AgentCampaignsQuery 代理活動查詢參數 (Domain Value Object)
type AgentCampaignsQuery struct {
	MerchantID       uint64   `json:"merchant_id"`
	Status           []string `json:"status"`
	TargetType       string   `json:"target_type"`
	CreatedBy        string   `json:"created_by"`
	CreatedStartAt   string   `json:"created_start_at"`
	CreatedEndAt     string   `json:"created_end_at"`
	ScheduledStartAt string   `json:"scheduled_start_at"`
	ScheduledEndAt   string   `json:"scheduled_end_at"`
	Page             int      `json:"page"`
	PageSize         int      `json:"page_size"`
	OrderBy          string   `json:"order_by"`
	OrderDir         string   `json:"order_dir"`
	IncludeTotal     bool     `json:"include_total"`
}

// AgentMessagesQuery 代理訊息查詢參數 (Domain Value Object)
type AgentMessagesQuery struct {
	AgentID      uint64 `json:"agent_id"`
	MerchantID   uint64 `json:"merchant_id"`
	IsRead       *bool  `json:"is_read"`
	Page         int    `json:"page"`
	PageSize     int    `json:"page_size"`
	OrderBy      string `json:"order_by"`
	OrderDir     string `json:"order_dir"`
	IncludeTotal bool   `json:"include_total"`
}

// MessageCampaignsQuery 會員訊息活動查詢參數 (Domain Value Object)
type MessageCampaignsQuery struct {
	MerchantID     uint64   `json:"merchant_id"`
	Category       string   `json:"category"`
	Item           string   `json:"item"`
	Status         []string `json:"status"`
	IncludeDeleted bool     `json:"include_deleted"`
	ShowAutoSend   bool     `json:"show_auto_send"`
	ObjectType     string   `json:"object_type"`
	CreatedBy      string   `json:"created_by"`
	StartAt        string   `json:"start_at"`
	EndAt          string   `json:"end_at"`
	Page           int      `json:"page"`
	PageSize       int      `json:"page_size"`
	OrderBy        string   `json:"order_by"`
	OrderDir       string   `json:"order_dir"`
	IncludeTotal   bool     `json:"include_total"`
	Focus          string   `json:"focus"`
}

// Validate 驗證代理活動查詢參數
func (q *AgentCampaignsQuery) Validate() error {
	if q.MerchantID == 0 {
		return ErrMerchantIDRequired
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	if q.OrderBy == "" {
		q.OrderBy = "created_at"
	}
	if q.OrderDir == "" {
		q.OrderDir = "desc"
	}
	return nil
}

// Limit 計算分頁限制數量
func (q *AgentCampaignsQuery) Limit() int {
	return q.PageSize
}

// Offset 計算分頁偏移量
func (q *AgentCampaignsQuery) Offset() int {
	return (q.Page - 1) * q.PageSize
}

// Validate 驗證代理訊息查詢參數
func (q *AgentMessagesQuery) Validate() error {
	if q.AgentID == 0 {
		return ErrAgentIDRequired
	}
	if q.MerchantID == 0 {
		return ErrMerchantIDRequired
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	if q.OrderBy == "" {
		q.OrderBy = "created_at"
	}
	if q.OrderDir == "" {
		q.OrderDir = "desc"
	}
	return nil
}

// Limit 計算分頁限制數量
func (q *AgentMessagesQuery) Limit() int {
	return q.PageSize
}

// Offset 計算分頁偏移量
func (q *AgentMessagesQuery) Offset() int {
	return (q.Page - 1) * q.PageSize
}

// Validate 驗證會員訊息活動查詢參數
func (q *MessageCampaignsQuery) Validate() error {
	if q.MerchantID == 0 {
		return ErrMerchantIDRequired
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	if q.OrderBy == "" {
		q.OrderBy = "created_at"
	}
	if q.OrderDir == "" {
		q.OrderDir = "desc"
	}
	return nil
}

// Limit 計算分頁限制數量
func (q *MessageCampaignsQuery) Limit() int {
	return q.PageSize
}

// Offset 計算分頁偏移量
func (q *MessageCampaignsQuery) Offset() int {
	return (q.Page - 1) * q.PageSize
}
