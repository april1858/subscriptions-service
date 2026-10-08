package dto

//import "github.com/go-playground/validator/v10"

type CreateSubscriptionRequest struct {
	ServiceName string `json:"service_name" validate:"required,min=1,max=255"`
	Price       int    `json:"price" validate:"gte=0"`
	UserID      string `json:"user_id" validate:"required,min=1,max=255"`
	StartDate   string `json:"start_date" validate:"required,min=1"`
}
