package dto

import "time"

// CheckoutRequest contains the request body for creating a checkout session
type CheckoutRequest struct {
	PriceID string `json:"priceId"`
}

// SubscriptionInfoResponse represents the user's subscription status
type SubscriptionInfoResponse struct {
	IsPremium          bool       `json:"isPremium"`
	SubscriptionStatus string     `json:"subscriptionStatus"`
	CurrentPeriodEnd   *time.Time `json:"currentPeriodEnd,omitempty"`
	CancelAtPeriodEnd  bool       `json:"cancelAtPeriodEnd"`
	ProfileFlair       string     `json:"profileFlair"`
}

// CheckoutResponse contains the Stripe checkout session URL
type CheckoutResponse struct {
	CheckoutURL string `json:"checkoutUrl"`
}

// BillingHistoryEntry represents a single billing event
type BillingHistoryEntry struct {
	ID          string `json:"id"`
	Date        string `json:"date"`
	Description string `json:"description"`
	Amount      string `json:"amount"`
	Status      string `json:"status"`
	InvoiceURL  string `json:"invoiceUrl,omitempty"`
}

// BillingHistoryResponse contains the user's billing history
type BillingHistoryResponse struct {
	Data []BillingHistoryEntry `json:"data"`
}
