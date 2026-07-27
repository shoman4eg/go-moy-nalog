package moynalog

// IncomeType is the counterparty class a receipt is issued to.
type IncomeType string

// Supported income types.
const (
	IncomeTypeIndividual    IncomeType = "FROM_INDIVIDUAL"
	IncomeTypeLegalEntity   IncomeType = "FROM_LEGAL_ENTITY"
	IncomeTypeForeignAgency IncomeType = "FROM_FOREIGN_AGENCY"
)

// Valid reports whether t is a known income type.
func (t IncomeType) Valid() bool {
	switch t {
	case IncomeTypeIndividual, IncomeTypeLegalEntity, IncomeTypeForeignAgency:
		return true
	default:
		return false
	}
}

// PaymentType is how a receipt was paid for.
type PaymentType string

// Supported payment types.
const (
	PaymentTypeCash    PaymentType = "CASH"
	PaymentTypeAccount PaymentType = "ACCOUNT"
)

// Valid reports whether t is a known payment type.
func (t PaymentType) Valid() bool {
	switch t {
	case PaymentTypeCash, PaymentTypeAccount:
		return true
	default:
		return false
	}
}

// BuyerType filters an income listing by counterparty class.
type BuyerType string

// Supported buyer types.
const (
	BuyerTypePerson        BuyerType = "PERSON"
	BuyerTypeCompany       BuyerType = "COMPANY"
	BuyerTypeForeignAgency BuyerType = "FOREIGN_AGENCY"
)

// Valid reports whether t is a known buyer type.
func (t BuyerType) Valid() bool {
	switch t {
	case BuyerTypePerson, BuyerTypeCompany, BuyerTypeForeignAgency:
		return true
	default:
		return false
	}
}

// ReceiptType filters an income listing by receipt state.
type ReceiptType string

// Supported receipt types.
const (
	ReceiptTypeRegistered ReceiptType = "REGISTERED"
	ReceiptTypeCancelled  ReceiptType = "CANCELLED"
)

// Valid reports whether t is a known receipt type.
func (t ReceiptType) Valid() bool {
	switch t {
	case ReceiptTypeRegistered, ReceiptTypeCancelled:
		return true
	default:
		return false
	}
}

// CancelComment is the reason a receipt is cancelled for. The API only accepts
// these two exact strings.
type CancelComment string

// Supported cancellation reasons.
const (
	// CancelCommentMistake — "Чек сформирован ошибочно".
	CancelCommentMistake CancelComment = "Чек сформирован ошибочно"
	// CancelCommentRefund — "Возврат средств".
	CancelCommentRefund CancelComment = "Возврат средств"
)

// Valid reports whether c is a known cancellation reason.
func (c CancelComment) Valid() bool {
	switch c {
	case CancelCommentMistake, CancelCommentRefund:
		return true
	default:
		return false
	}
}

// SortBy orders an income listing.
type SortBy string

// Supported sort orders.
const (
	SortByOperationTimeDesc SortBy = "operation_time:desc"
	SortByOperationTimeAsc  SortBy = "operation_time:asc"
	SortByTotalAmountDesc   SortBy = "total_amount:desc"
	SortByTotalAmountAsc    SortBy = "total_amount:asc"
)

// Valid reports whether s is a known sort order.
func (s SortBy) Valid() bool {
	switch s {
	case SortByOperationTimeDesc, SortByOperationTimeAsc, SortByTotalAmountDesc, SortByTotalAmountAsc:
		return true
	default:
		return false
	}
}
