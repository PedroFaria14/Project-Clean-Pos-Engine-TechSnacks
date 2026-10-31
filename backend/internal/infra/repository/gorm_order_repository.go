package repository

import (
	"context"
	"errors"
	"fmt"

	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/domain/order"
	"PedroFaria14/clean-pos-engine/internal/domain/product"
	"PedroFaria14/clean-pos-engine/internal/domain/shared"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrderModel struct {
	ID         string `gorm:"primaryKey"`
	TotalCents int64
	Status     string
	CreatedBy  *string
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
	return &GormOrderRepository{db: db}
}

func orderModel(o *order.Order) OrderModel {
	model := OrderModel{
		ID:         o.ID(),
		TotalCents: o.CalculateTotal().Cents(),
		Status:     string(o.Status()),
	}
	if creator := o.CreatorID(); creator != "" {
		model.CreatedBy = &creator
	}

	for _, item := range o.Cart().Items() {
		model.Items = append(model.Items, OrderItemModel{
			ProductID:   item.Product().ID(),
			ProductName: item.Product().Name(),
			Quantity:    item.Quantity(),
			PriceCents:  item.Product().Price().Cents(),
		})
	}

	return model
}

func uniqueViolation(err error) bool {
	var sqlErr interface{ SQLState() string }
	return errors.As(err, &sqlErr) && sqlErr.SQLState() == "23505"
}

func (r *GormOrderRepository) Save(o *order.Order) error {
	model := orderModel(o)
	err := r.db.Create(&model).Error
	if uniqueViolation(err) {
		return port.ErrOrderConflict
	}
	return err
}

func (r *GormOrderRepository) FindByID(id string) (*order.Order, error) {
	var model OrderModel
	err := r.db.Preload("Items").Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, port.ErrOrderNotFound
		}
		return nil, err
	}

	return restoreOrderModel(model)
}

func restoreOrderModel(model OrderModel) (*order.Order, error) {
	items := make([]order.CartItem, 0, len(model.Items))
	for _, item := range model.Items {
		price, err := shared.NewMoney(item.PriceCents)
		if err != nil {
			return nil, fmt.Errorf("restore order item price: %w", err)
		}
		p, err := product.NewProduct(item.ProductID, item.ProductName, price)
		if err != nil {
			return nil, fmt.Errorf("restore order item product: %w", err)
		}
		cartItem, err := order.NewCartItem(p, item.Quantity)
		if err != nil {
			return nil, fmt.Errorf("restore order item: %w", err)
		}
		items = append(items, *cartItem)
	}
	total, err := shared.NewMoney(model.TotalCents)
	if err != nil {
		return nil, fmt.Errorf("restore order total: %w", err)
	}
	creator := ""
	if model.CreatedBy != nil {
		creator = *model.CreatedBy
	}
	return order.RestoreOrderWithCreator(model.ID, order.Status(model.Status), items, total, creator)
}

type checkoutRecordModel struct {
	port.CheckoutRecord `gorm:"embedded"`
}

func (checkoutRecordModel) TableName() string { return "checkout_idempotency_records" }

func (r *GormOrderRepository) FindCheckout(ctx context.Context, key string) (port.CheckoutRecord, error) {
	var model checkoutRecordModel
	err := r.db.WithContext(ctx).Where("key = ?", key).Take(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return port.CheckoutRecord{}, port.ErrCheckoutNotFound
	}
	return model.CheckoutRecord, err
}

var errCheckoutAlreadyExists = errors.New("checkout already committed")

func validateCheckoutSnapshot(o *order.Order, record port.CheckoutRecord) error {
	if record.Key == "" || len(record.RequestHash) != 64 || record.OrderID != o.ID() ||
		record.TotalCents != o.CalculateTotal().Cents() || record.Status != string(order.StatusAwaitingPayment) ||
		o.Status() != order.StatusAwaitingPayment {
		return errors.New("invalid checkout snapshot")
	}
	return nil
}

func (r *GormOrderRepository) SaveCheckout(ctx context.Context, o *order.Order, record port.CheckoutRecord) (port.CheckoutRecord, error) {
	if err := validateCheckoutSnapshot(o, record); err != nil {
		return port.CheckoutRecord{}, err
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := orderModel(o)
		if err := tx.Create(&model).Error; err != nil {
			return err
		}
		result := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "key"}}, DoNothing: true,
		}).Create(&checkoutRecordModel{CheckoutRecord: record})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errCheckoutAlreadyExists
		}
		return nil
	})
	if err == nil {
		return record, nil
	}
	if !errors.Is(err, errCheckoutAlreadyExists) && !uniqueViolation(err) {
		return port.CheckoutRecord{}, err
	}
	// The losing transaction (including its candidate order/items) has rolled back.
	existing, lookupErr := r.FindCheckout(ctx, record.Key)
	if lookupErr == nil {
		if existing.RequestHash != record.RequestHash {
			return port.CheckoutRecord{}, port.ErrIdempotencyConflict
		}
		return existing, nil
	}
	if errors.Is(lookupErr, port.ErrCheckoutNotFound) && uniqueViolation(err) {
		return port.CheckoutRecord{}, port.ErrOrderConflict
	}
	return port.CheckoutRecord{}, lookupErr
}
