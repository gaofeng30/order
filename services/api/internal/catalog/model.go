package catalog

// Category is one visible catalog category and its visible products.
type Category struct {
	ID       uint64
	Name     string
	Products []Product
}

// Product is the public catalog subset of one visible product.
type Product struct {
	ID            uint64
	CategoryID    uint64
	Name          string
	Description   string
	Specification string
	PriceCents    uint32
}
