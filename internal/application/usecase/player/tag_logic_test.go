package player

import (
	"context"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/stretchr/testify/require"
)

func TestPlayerTagUseCase_tagNeedsUpdate(t *testing.T) {
	usecase := &PlayerTagUseCase{}

	tests := []struct {
		name         string
		existing     *entity.Tag
		incoming     *entity.Tag
		shouldUpdate bool
		description  string
	}{
		{
			name:         "name changed",
			existing:     &entity.Tag{Name: "Old Name"},
			incoming:     &entity.Tag{Name: "New Name"},
			shouldUpdate: true,
			description:  "名稱變更應該需要更新",
		},
		{
			name: "same name but newer update time",
			existing: &entity.Tag{
				Name:      "Same Name",
				UpdatedAt: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			incoming: &entity.Tag{
				Name:      "Same Name",
				UpdatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			shouldUpdate: false,
			description:  "相同名稱時不應該更新（當前實作只檢查名稱）",
		},
		{
			name:     "same name but deleted status changed",
			existing: &entity.Tag{Name: "Same Name"},
			incoming: func() *entity.Tag {
				tag := &entity.Tag{Name: "Same Name"}
				now := time.Now()
				tag.DeletedAt = &now
				return tag
			}(),
			shouldUpdate: false,
			description:  "相同名稱時不應該更新（當前實作只檢查名稱）",
		},
		{
			name: "no changes",
			existing: &entity.Tag{
				Name:      "Same Name",
				UpdatedAt: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			incoming: &entity.Tag{
				Name:      "Same Name",
				UpdatedAt: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			shouldUpdate: false,
			description:  "無變更時不應該需要更新",
		},
		{
			name: "older update time",
			existing: &entity.Tag{
				Name:      "Same Name",
				UpdatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			incoming: &entity.Tag{
				Name:      "Same Name",
				UpdatedAt: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			shouldUpdate: false,
			description:  "更新時間較舊不應該需要更新",
		},
		{
			name:         "deleted status changed - both nil",
			existing:     &entity.Tag{Name: "Same Name"},
			incoming:     &entity.Tag{Name: "Same Name"},
			shouldUpdate: false,
			description:  "兩者都沒有刪除時間不應該更新",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := usecase.tagNeedsUpdate(tt.existing, tt.incoming)
			require.Equal(t, tt.shouldUpdate, result, tt.description)
		})
	}
}

func TestFilterTagsNeedingUpdate_EmptyInput(t *testing.T) {
	usecase := &PlayerTagUseCase{}

	result, err := usecase.filterTagsNeedingUpdate(context.TODO(), []*entity.Tag{})

	require.NoError(t, err)
	require.Empty(t, result)
}
