package model

// AsaasPaymentsListResponse models the paginated list response from Asaas /v3/payments.
type AsaasPaymentsListResponse struct {
	Object     string                 `json:"object"` // "list"
	HasMore    bool                   `json:"hasMore"`
	TotalCount int32                  `json:"totalCount"`
	Limit      int32                  `json:"limit"`
	Offset     int32                  `json:"offset"`
	Data       []AsaasPaymentResponse `json:"data"`
}
