package menu

// Category is one visible category containing only products for the selected meal.
type Category struct {
	ID       uint64
	Name     string
	Products []Product
}

// Product is the normal-price public menu subset plus its selected-date sold-out fact.
type Product struct {
	ID            uint64
	CategoryID    uint64
	Name          string
	Description   string
	Specification string
	PriceCents    uint32
	SoldOut       bool
}
