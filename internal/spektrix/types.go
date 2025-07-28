package spektrix

import "time"

// Customer represents a Spektrix customer
type Customer struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// CreateCustomerRequest for creating new customers
type CreateCustomerRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

// Address represents a customer address
type Address struct {
	Country  string `json:"country"`
	Postcode string `json:"postcode"`
	Line1    string `json:"line1,omitempty"`
	Line2    string `json:"line2,omitempty"`
	City     string `json:"city,omitempty"`
	State    string `json:"state,omitempty"`
}

// Tag represents a Spektrix tag
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TagReference for tag operations
type TagReference struct {
	ID string `json:"id"`
}

// APIError represents Spektrix API error response
type APIError struct {
	Message   string `json:"message"`
	Code      int    `json:"code"`
	Timestamp time.Time
}

func (e APIError) Error() string {
	return e.Message
}
