package postgres

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BacktestPreset struct {
	ID        string
	UserID    string
	Name      string
	Config    []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

type BacktestPresetStore struct {
	db *gorm.DB
}

func NewBacktestPresetStore(db *gorm.DB) *BacktestPresetStore {
	return &BacktestPresetStore{db: db}
}

func (s *BacktestPresetStore) Save(ctx context.Context, userID string, config []byte) error {
	_, err := s.SaveNamed(ctx, userID, "default", config)
	return err
}

func (s *BacktestPresetStore) SaveNamed(ctx context.Context, userID, name string, config []byte) (string, error) {
	m := BacktestPresetModel{
		UserID: userID,
		Name:   name,
		Config: config,
	}

	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"config", "updated_at"}),
	}).Create(&m).Error

	if err != nil {
		return "", err
	}

	// If we need the ID after create/upsert
	return m.ID, nil
}

func (s *BacktestPresetStore) Load(ctx context.Context, userID string) ([]byte, error) {
	var m BacktestPresetModel
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		First(&m).Error

	if err != nil {
		return nil, err
	}
	return m.Config, nil
}

func (s *BacktestPresetStore) List(ctx context.Context, userID string) ([]BacktestPreset, error) {
	var models []BacktestPresetModel
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&models).Error

	if err != nil {
		return nil, err
	}

	out := make([]BacktestPreset, len(models))
	for i, m := range models {
		out[i] = BacktestPreset{
			ID:        m.ID,
			UserID:    m.UserID,
			Name:      m.Name,
			Config:    m.Config,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		}
	}
	return out, nil
}

func (s *BacktestPresetStore) Delete(ctx context.Context, userID, id string) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&BacktestPresetModel{})
	
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *BacktestPresetStore) SeedDefaults(ctx context.Context) error {
	// no-op
	return nil
}

// NotFound 判斷是否為未找到錯誤。
func (s *BacktestPresetStore) NotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
