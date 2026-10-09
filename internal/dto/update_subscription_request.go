package dto

type UpdateSubscriptionRequest struct {
	ServiceName *string `json:"service_name,omitempty" validate:"omitempty,min=1,max=255"`
	Price       *int    `json:"price,omitempty" validate:"omitempty,gte=0"`
	StartDate   *string `json:"start_date,omitempty" validate:"omitempty,min=1"`
}
