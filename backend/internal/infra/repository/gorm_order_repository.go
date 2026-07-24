package repository

import (
	"PedroFaria14/clean-pos-engine/internal/domain/order"
	"PedroFaria14/clean-pos-engine/internal/domain/product"
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
	"errors"

	"gorm.io/gorm"
)

type OrderModel struct {
	ID         string `gorm:"primaryKey"`
	TotalCents int64
	Items      []OrderItemModel `gorm:"foreignKey:OrderID"`
}

type OrderItemModel struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	OrderID     string // Chave Estrangeira
	ProductID   string
	ProductName string
	Quantity    int
	PriceCents  int64
}

type GormOrderRepository struct {
	db *gorm.DB
}

func NewGormOrderRepository(db *gorm.DB) *GormOrderRepository {
	db.AutoMigrate(&OrderModel{}, &OrderItemModel{})
	return &GormOrderRepository{db: db}
}

func (r *GormOrderRepository) Save(o *order.Order) error {
	model := OrderModel{
		ID:         o.ID(),
		TotalCents: o.CalculateTotal().Cents(),
	}

	for _, item := range o.Cart().Items() {
		model.Items = append(model.Items, OrderItemModel{
			ProductID:   item.Product().ID(),
			ProductName: item.Product().Name(),
			Quantity:    item.Quantity(),
			PriceCents:  item.Product().Price().Cents(),
		})
	}

	return r.db.Create(&model).Error
}

func (r *GormOrderRepository) FindByID(id string) (*order.Order, error) {
	var model OrderModel
	err := r.db.Preload("Items").Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}

	domainOrder := order.NewOrder(model.ID)
	for _, item := range model.Items {
		price, _ := shared.NewMoney(item.PriceCents)
		p, _ := product.NewProduct(item.ProductID, item.ProductName, price)

		cartItem, _ := order.NewCartItem(p, item.Quantity)

		domainOrder.Cart().AddItem(*cartItem)
	}

	return domainOrder, nil
}
