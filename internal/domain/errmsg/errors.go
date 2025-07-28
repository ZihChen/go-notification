package errmsg

import "errors"

var (
	ErrMerchantNotFound       = errors.New("merchant not found")
	ErrDeleteMerchantNotFound = errors.New("delete merchant not found")
	ErrManagerNotFound        = errors.New("manager not found")
	ErrDeleteManagerNotFound  = errors.New("delete manager not found")
	ErrPlayerNotFound         = errors.New("player not found")
	ErrDeletePlayerNotFound   = errors.New("delete player not found")
)
