package repository

import (
	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/domain/product"
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProductModel struct {
	ID    string `gorm:"primaryKey"`
	Name  string
	Price int64
}

func (ProductModel) TableName() string {
	return "products"
}

type GormProductRepository struct {
	db *gorm.DB
}

func NewGormProductRepository(db *gorm.DB) *GormProductRepository {
	return &GormProductRepository{db: db}
}

func (r *GormProductRepository) Save(p *product.Product) error {
	model := ProductModel{
		ID:    p.ID(),
		Name:  p.Name(),
		Price: p.Price().Cents(),
	}
	err := r.db.Create(&model).Error
	if uniqueViolation(err) {
		return port.ErrProductConflict
	}
	return err
}

func (r *GormProductRepository) FindByID(id string) (*product.Product, error) {
	var model ProductModel
	err := r.db.Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, port.ErrProductNotFound
		}
		return nil, err
	}

	price, err := shared.NewMoney(model.Price)
	if err != nil {
		return nil, err
	}
	return product.NewProduct(model.ID, model.Name, price)
}

func (r *GormProductRepository) FindAll() ([]*product.Product, error) {
	var models []ProductModel
	err := r.db.Order("name ASC, id ASC").Find(&models).Error
	if err != nil {
		return nil, err
	}

	var domainProducts []*product.Product
	for _, m := range models {
		price, err := shared.NewMoney(m.Price)
		if err != nil {
			return nil, err
		}
		p, err := product.NewProduct(m.ID, m.Name, price)
		if err != nil {
			return nil, err
		}
		domainProducts = append(domainProducts, p)
	}
	return domainProducts, nil
}

func (r *GormProductRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&ProductModel{}).Error
}

func (r *GormProductRepository) Update(ctx context.Context, id string, name *string, price *int64) (*product.Product, error) {
	fields := map[string]any{}
	if name != nil {
		fields["name"] = *name
	}
	if price != nil {
		fields["price"] = *price
	}
	var model ProductModel
	result := r.db.WithContext(ctx).Model(&model).Clauses(clause.Returning{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, port.ErrProductNotFound
	}
	money, err := shared.NewMoney(model.Price)
	if err != nil {
		return nil, err
	}
	return product.NewProduct(model.ID, model.Name, money)
}
