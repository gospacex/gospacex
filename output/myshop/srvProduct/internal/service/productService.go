package service

import (
	"context"

	"myshop/srvProduct/internal/model"
	"myshop/srvProduct/internal/repository"
)

type ProductService struct { repo *repository.ProductRepo }

func NewProductService(repo *repository.ProductRepo) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) Create(ctx context.Context, m *model.Product) error {
	return s.repo.Create(ctx, m)
}

func (s *ProductService) Get(ctx context.Context, id int64) (*model.Product, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProductService) List(ctx context.Context) ([]*model.Product, error) {
	return s.repo.List(ctx)
}

func (s *ProductService) Update(ctx context.Context, m *model.Product) error {
	return s.repo.Update(ctx, m)
}

func (s *ProductService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
