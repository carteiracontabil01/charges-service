package model

// AsaasUpdateChargeRequest is the payload to update an existing charge (payment) in Asaas.
// According to Asaas docs: only charges awaiting payment or overdue can be updated.
// Once created, the customer cannot be changed.
//
// IMPORTANT (product rule):
// We only allow updating: value, dueDate, and financial settings (interest/fine/discount).
// https://docs.asaas.com/reference/atualizar-cobranca-existente
type AsaasUpdateChargeRequest struct {
	Value                                      *float64          `json:"value,omitempty"`
	DueDate                                    *string           `json:"dueDate,omitempty"` // YYYY-MM-DD
	Discount      *AsaasChargeDiscount `json:"discount,omitempty"`
	Interest      *AsaasChargeInterest `json:"interest,omitempty"`
	Fine          *AsaasChargeFine     `json:"fine,omitempty"`
}
