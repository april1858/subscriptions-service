package dto

import "time"

type SubscriptionCostRequest struct {
	UserID      *string   `json:"user_id,omitempty"`      // nullable: если не передано — все пользователи
	ServiceName *string   `json:"service_name,omitempty"` // nullable: если не передано — все сервисы
	From        time.Time `json:"from"`                   // начало периода (включительно)
	To          time.Time `json:"to"`                     // конец периода (включительно)
}

type SubscriptionCostResponse struct {
	TotalAmount int    `json:"total_amount"`
	Currency    string `json:"currency"` // можно сделать фиксированным или брать из БД
	PeriodFrom  string `json:"period_from"`
	PeriodTo    string `json:"period_to"`
	Filters     struct {
		UserID      string `json:"user_id,omitempty"`
		ServiceName string `json:"service_name,omitempty"`
	} `json:"filters"`
}
