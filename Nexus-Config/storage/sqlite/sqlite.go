package sqlite

import (
	"context"
	"crypto/md5"
	"fmt"
	"time"

	"github.com/krustd/nexus-config/common"
	"github.com/krustd/nexus-config/storage"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type sqliteStorage struct {
	db *gorm.DB
}

// NewSQLiteStorage 创建 SQLite 存储实例
func NewSQLiteStorage(filePath string) (storage.Storage, error) {
	db, err := gorm.Open(sqlite.Open(filePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite failed: %w", err)
	}

	return &sqliteStorage{db: db}, nil
}

func (s *sqliteStorage) Init(ctx context.Context) error {
	// 自动迁移表结构
	return s.db.WithContext(ctx).AutoMigrate(
		&common.ConfigNamespace{},
		&common.ConfigItem{},
		&common.GrayRule{},
	)
}

func (s *sqliteStorage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// === Namespace 操作 ===

func (s *sqliteStorage) CreateNamespace(ctx context.Context, ns *common.ConfigNamespace) error {
	ns.CreatedAt = time.Now()
	ns.UpdatedAt = time.Now()
	return s.db.WithContext(ctx).Create(ns).Error
}

func (s *sqliteStorage) GetNamespace(ctx context.Context, id string) (*common.ConfigNamespace, error) {
	var ns common.ConfigNamespace
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&ns).Error
	if err != nil {
		return nil, err
	}
	return &ns, nil
}

func (s *sqliteStorage) ListNamespaces(ctx context.Context) ([]*common.ConfigNamespace, error) {
	var list []*common.ConfigNamespace
	err := s.db.WithContext(ctx).Find(&list).Error
	return list, err
}

func (s *sqliteStorage) DeleteNamespace(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除命名空间下的所有配置
		if err := tx.Where("namespace = ?", id).Delete(&common.ConfigItem{}).Error; err != nil {
			return err
		}
		// 删除命名空间下的所有灰度规则
		if err := tx.Where("namespace = ?", id).Delete(&common.GrayRule{}).Error; err != nil {
			return err
		}
		// 删除命名空间
		return tx.Where("id = ?", id).Delete(&common.ConfigNamespace{}).Error
	})
}

// === ConfigItem 操作 ===

func (s *sqliteStorage) SaveDraft(ctx context.Context, namespace, key string, value string, format common.ConfigFormat) error {
	draftMD5 := fmt.Sprintf("%x", md5.Sum([]byte(value)))

	// 检查是否已存在
	var item common.ConfigItem
	err := s.db.WithContext(ctx).Where("namespace = ? AND key = ?", namespace, key).First(&item).Error

	if err == gorm.ErrRecordNotFound {
		// 新建
		item = common.ConfigItem{
			Namespace:  namespace,
			Key:        key,
			Format:     format,
			DraftValue: value,
			DraftMD5:   draftMD5,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		return s.db.WithContext(ctx).Create(&item).Error
	}

	if err != nil {
		return err
	}

	// 更新草稿
	return s.db.WithContext(ctx).Model(&item).Updates(map[string]interface{}{
		"draft_value": value,
		"draft_md5":   draftMD5,
		"format":      format,
		"updated_at":  time.Now(),
	}).Error
}

func (s *sqliteStorage) GetDraft(ctx context.Context, namespace, key string) (*common.ConfigItem, error) {
	var item common.ConfigItem
	err := s.db.WithContext(ctx).Where("namespace = ? AND key = ?", namespace, key).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *sqliteStorage) PublishConfig(ctx context.Context, namespace, key string) error {
	var item common.ConfigItem
	err := s.db.WithContext(ctx).Where("namespace = ? AND key = ?", namespace, key).First(&item).Error
	if err != nil {
		return err
	}

	now := time.Now()
	return s.db.WithContext(ctx).Model(&item).Updates(map[string]interface{}{
		"published_value": item.DraftValue,
		"published_md5":   item.DraftMD5,
		"published_at":    &now,
		"updated_at":      now,
	}).Error
}

func (s *sqliteStorage) GetPublishedConfig(ctx context.Context, namespace, key string) (*common.ConfigItem, error) {
	var item common.ConfigItem
	err := s.db.WithContext(ctx).Where("namespace = ? AND key = ? AND published_value != ''", namespace, key).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *sqliteStorage) ListConfigs(ctx context.Context, namespace string) ([]*common.ConfigItem, error) {
	var list []*common.ConfigItem
	err := s.db.WithContext(ctx).Where("namespace = ?", namespace).Find(&list).Error
	return list, err
}

func (s *sqliteStorage) DeleteConfig(ctx context.Context, namespace, key string) error {
	return s.db.WithContext(ctx).Where("namespace = ? AND key = ?", namespace, key).Delete(&common.ConfigItem{}).Error
}

// === GrayRule 操作 ===

func (s *sqliteStorage) SaveGrayRule(ctx context.Context, rule *common.GrayRule) error {
	// 检查是否已存在
	var existing common.GrayRule
	err := s.db.WithContext(ctx).Where("namespace = ? AND key = ?", rule.Namespace, rule.Key).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 新建
		rule.CreatedAt = time.Now()
		rule.UpdatedAt = time.Now()
		return s.db.WithContext(ctx).Create(rule).Error
	}

	if err != nil {
		return err
	}

	// 更新
	return s.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
		"percentage": rule.Percentage,
		"enabled":    rule.Enabled,
		"updated_at": time.Now(),
	}).Error
}

func (s *sqliteStorage) GetGrayRule(ctx context.Context, namespace, key string) (*common.GrayRule, error) {
	var rule common.GrayRule
	err := s.db.WithContext(ctx).Where("namespace = ? AND key = ?", namespace, key).First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *sqliteStorage) DeleteGrayRule(ctx context.Context, namespace, key string) error {
	return s.db.WithContext(ctx).Where("namespace = ? AND key = ?", namespace, key).Delete(&common.GrayRule{}).Error
}

func (s *sqliteStorage) ListGrayRules(ctx context.Context, namespace string) ([]*common.GrayRule, error) {
	var list []*common.GrayRule
	err := s.db.WithContext(ctx).Where("namespace = ?", namespace).Find(&list).Error
	return list, err
}
