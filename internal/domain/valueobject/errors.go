package valueobject

import "errors"

// Domain層錯誤定義 - Agent Campaign相關
var (
	ErrTitleRequired         = errors.New("title is required")
	ErrContentRequired       = errors.New("content is required") 
	ErrTargetTypeRequired    = errors.New("target_type is required")
	ErrStatusRequired        = errors.New("status is required")
	ErrCreatedByRequired     = errors.New("created_by is required")
	ErrTargetDetailsRequired = errors.New("target_details is required for specified target_type")
	ErrInvalidStatus         = errors.New("invalid status: must be 'draft' or 'scheduled'")
	ErrInvalidTargetType     = errors.New("invalid target_type: must be 'all', 'specific', or 'line'")
)