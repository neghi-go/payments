package payments

type Payments struct{}

type Option func(*Payments)

func New(opts ...Option) *Payments {
	return &Payments{}
}
