package repository

import (
	"context"

	"myshop/pkg/database"
	"myshop/srvProduct/internal/model"

	"gorm.io/gorm"
)

type ProductRepo struct { db *gorm.DB }

func NewProductRepo(db *gorm.DB) *ProductRepo {
	if db == nil {
		db, _ = database.NewDB(nil)
	}
	return &ProductRepo{db: db}
}

func (r *ProductRepo) Create(ctx context.Context, m *model.Product) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *ProductRepo) GetByID(ctx context.Context, id int64) (*model.Product, error) {
	var m model.Product
	if err := r.db.WithContext(ctx).Where("id = ? AND is_del = 0", id).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ProductRepo) List(ctx context.Context) ([]*model.Product, error) {
	var list []*model.Product
	return list, r.db.WithContext(ctx).Where("is_del = 0").Find(&list).Error
}

func (r *ProductRepo) Update(ctx context.Context, m *model.Product) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *ProductRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&model.Product{}).Where("id = ?", id).Update("is_del", 1).Error
}
