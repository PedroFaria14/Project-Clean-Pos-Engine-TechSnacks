package repository

import (
	"PedroFaria14/clean-pos-engine/internal/domain/product"
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
	"errors"

	"gorm.io/gorm"
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
	db.AutoMigrate(&ProductModel{})
	return &GormProductRepository{db: db}
}

func (r *GormProductRepository) Save(p *product.Product) error {
	model := ProductModel{
		ID:    p.ID(),
		Name:  p.Name(),
		Price: p.Price().Cents(),
	}
	return r.db.Create(&model).Error
}

func (r *GormProductRepository) FindByID(id string) (*product.Product, error) {
	var model ProductModel
	err := r.db.Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	price, _ := shared.NewMoney(model.Price)
	return product.NewProduct(model.ID, model.Name, price)
}

func (r *GormProductRepository) FindAll() ([]*product.Product, error) {
	var models []ProductModel
	err := r.db.Find(&models).Error
	if err != nil {
		return nil, err
	}

	var domainProducts []*product.Product
	for _, m := range models {
		price, _ := shared.NewMoney(m.Price)
		p, _ := product.NewProduct(m.ID, m.Name, price)
		domainProducts = append(domainProducts, p)
	}
	return domainProducts, nil
}

func (r *GormProductRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&ProductModel{}).Error
}
