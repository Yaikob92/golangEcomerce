package services

import (
	"context"
	"fmt"
	"log/slog"

	"backend/dto"
	"backend/repositories"
)

type OrderService interface {
	List(ctx context.Context, search, status, sort string, page, limit int) (dto.OrderListResponse, error)
	GetByID(ctx context.Context, id string) (dto.OrderResponse, error)
}

type orderService struct {
	orderRepo repositories.OrderRepository
}

func NewOrderService(orderRepo repositories.OrderRepository) OrderService {
	return &orderService{orderRepo: orderRepo}
}

func (s *orderService) List(ctx context.Context, search, status, sort string, page, limit int) (dto.OrderListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	orders, total, err := s.orderRepo.List(ctx, search, status, sort, offset, limit)
	if err != nil {
		slog.Error("failed to list orders", "error", err)
		return dto.OrderListResponse{}, fmt.Errorf("failed to list orders: %w", err)
	}

	responses := make([]dto.OrderResponse, len(orders))
	for i, o := range orders {
		responses[i] = dto.OrderResponse{
			ID:            o.ID,
			OrderNumber:   o.OrderNumber,
			CustomerName:  o.CustomerName,
			Status:        o.Status,
			TotalAmount:   o.TotalAmount,
			PaymentStatus: o.PaymentStatus,
			CreatedAt:     o.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	if responses == nil {
		responses = []dto.OrderResponse{}
	}

	totalPages := 1
	if limit > 0 {
		totalPages = total / limit
		if total%limit != 0 {
			totalPages++
		}
	}

	return dto.OrderListResponse{
		Orders:     responses,
		TotalCount: total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (s *orderService) GetByID(ctx context.Context, id string) (dto.OrderResponse, error) {
	o, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return dto.OrderResponse{}, err
	}

	items, err := s.orderRepo.GetItemsByOrderID(ctx, id)
	if err != nil {
		slog.Warn("failed to fetch order items", "order_id", id, "error", err)
		items = nil
	}

	var itemResponses []dto.OrderItemResponse
	for _, item := range items {
		itemResponses = append(itemResponses, dto.OrderItemResponse{
			ID:          item.ID,
			ProductName: item.ProductName,
			ProductSKU:  item.ProductSKU,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			TotalPrice:  item.TotalPrice,
		})
	}
	if itemResponses == nil {
		itemResponses = []dto.OrderItemResponse{}
	}

	return dto.OrderResponse{
		ID:            o.ID,
		OrderNumber:   o.OrderNumber,
		CustomerName:  o.CustomerName,
		Status:        o.Status,
		Subtotal:      o.Subtotal,
		ShippingCost:  o.ShippingCost,
		Tax:           o.Tax,
		Discount:      o.Discount,
		TotalAmount:   o.TotalAmount,
		PaymentMethod: o.PaymentMethod,
		PaymentStatus: o.PaymentStatus,
		CreatedAt:     o.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     o.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Items:         itemResponses,
	}, nil
}
