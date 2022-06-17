package moynalog

type incomeType int64

const (
	Individual incomeType = iota + 1
	LegalEntity
	ForeignAgency
)

func (t incomeType) String() string {
	return [...]string{"FROM_INDIVIDUAL", "FROM_LEGAL_ENTITY", "FROM_FOREIGN_AGENCY"}[t-1]
}

type paymentType int64

const (
	Cash paymentType = iota + 1
	Account
)

func (t paymentType) String() string {
	return [...]string{"CASH", "ACCOUNT"}[t-1]
}
