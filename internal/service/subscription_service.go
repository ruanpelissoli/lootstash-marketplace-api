package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
	"github.com/stripe/stripe-go/v81"
	checkoutsession "github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/subscription"
	"github.com/stripe/stripe-go/v81/webhook"
)

// StripeConfig holds Stripe-specific configuration
type StripeConfig struct {
	SecretKey       string
	WebhookSecret   string
	PriceID         string
	SuccessURL      string
	CancelURL       string
	AllowedPriceIDs []string // List of allowed price IDs for geo-based pricing
}

// SubscriptionService handles premium subscription logic
type SubscriptionService struct {
	profileRepo     repository.ProfileRepository
	billingRepo     repository.BillingEventRepository
	transactionRepo repository.TransactionRepository
	wishlistRepo    repository.WishlistRepository
	listingRepo     repository.ListingRepository
	redis           *cache.RedisClient
	invalidator     *cache.Invalidator
	config          StripeConfig
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(
	profileRepo repository.ProfileRepository,
	billingRepo repository.BillingEventRepository,
	transactionRepo repository.TransactionRepository,
	wishlistRepo repository.WishlistRepository,
	listingRepo repository.ListingRepository,
	redis *cache.RedisClient,
	config StripeConfig,
) *SubscriptionService {
	stripe.Key = config.SecretKey
	return &SubscriptionService{
		profileRepo:     profileRepo,
		billingRepo:     billingRepo,
		transactionRepo: transactionRepo,
		wishlistRepo:    wishlistRepo,
		listingRepo:     listingRepo,
		redis:           redis,
		invalidator:     cache.NewInvalidator(redis),
		config:          config,
	}
}

// GetSubscriptionInfo returns the user's subscription status
func (s *SubscriptionService) GetSubscriptionInfo(ctx context.Context, userID string) (*dto.SubscriptionInfoResponse, error) {
	profile, err := s.profileRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.SubscriptionInfoResponse{
		IsPremium:          profile.IsPremium,
		SubscriptionStatus: profile.SubscriptionStatus,
		CurrentPeriodEnd:   profile.SubscriptionCurrentPeriodEnd,
		CancelAtPeriodEnd:  profile.CancelAtPeriodEnd,
		ProfileFlair:       profile.GetProfileFlair(),
		UsernameColor:      profile.GetUsernameColor(),
	}, nil
}

// CreateCheckoutSession creates a Stripe checkout session for premium subscription
func (s *SubscriptionService) CreateCheckoutSession(ctx context.Context, userID string, priceID string) (*dto.CheckoutResponse, error) {
	// Determine which price ID to use
	effectivePriceID := priceID
	if effectivePriceID == "" {
		effectivePriceID = s.config.PriceID
	}

	// Validate price ID against allowed list
	if !s.isAllowedPriceID(effectivePriceID) {
		return nil, ErrInvalidPriceID
	}

	profile, err := s.profileRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Create or reuse Stripe customer
	var customerID string
	if profile.StripeCustomerID != nil {
		customerID = *profile.StripeCustomerID
	} else {
		// Get email from auth.users table
		email, err := s.profileRepo.GetEmailByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user email: %w", err)
		}

		params := &stripe.CustomerParams{
			Params: stripe.Params{
				Metadata: map[string]string{
					"user_id": userID,
				},
			},
		}
		params.Email = stripe.String(email)
		c, err := customer.New(params)
		if err != nil {
			return nil, fmt.Errorf("failed to create stripe customer: %w", err)
		}
		customerID = c.ID
		profile.StripeCustomerID = &customerID
		if err := s.profileRepo.Update(ctx, profile); err != nil {
			return nil, err
		}
		_ = s.invalidator.InvalidateProfile(ctx, userID)
		_ = s.invalidator.InvalidateProfileDTO(ctx, userID)
	}

	// Create checkout session
	sessionParams := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(effectivePriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(s.config.SuccessURL),
		CancelURL:  stripe.String(s.config.CancelURL),
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"user_id": userID,
			},
		},
	}

	sess, err := checkoutsession.New(sessionParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return &dto.CheckoutResponse{
		CheckoutURL: sess.URL,
	}, nil
}

// isAllowedPriceID validates that the price ID is in the allowed list
func (s *SubscriptionService) isAllowedPriceID(priceID string) bool {
	// If no allowed list configured, only allow the default price ID
	if len(s.config.AllowedPriceIDs) == 0 {
		return priceID == s.config.PriceID
	}

	for _, allowed := range s.config.AllowedPriceIDs {
		if allowed == priceID {
			return true
		}
	}
	return false
}

// CancelSubscription cancels the user's subscription at the end of the billing period
func (s *SubscriptionService) CancelSubscription(ctx context.Context, userID string) error {
	profile, err := s.profileRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if profile.StripeSubscriptionID == nil {
		return ErrNotFound
	}

	_, err = subscription.Update(*profile.StripeSubscriptionID, &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	profile.CancelAtPeriodEnd = true
	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return err
	}
	_ = s.invalidator.InvalidateProfile(ctx, userID)
	_ = s.invalidator.InvalidateProfileDTO(ctx, userID)

	return nil
}

// billingEventDisplayNames maps Stripe event types to user-friendly names
var billingEventDisplayNames = map[string]string{
	"checkout.session.completed":      "Subscription Started",
	"customer.subscription.updated":   "Subscription Updated",
	"customer.subscription.deleted":   "Subscription Cancelled",
	"invoice.payment_succeeded":       "Payment Succeeded",
	"invoice.payment_failed":          "Payment Failed",
}

// billingEventStatuses maps Stripe event types to simple status labels
var billingEventStatuses = map[string]string{
	"checkout.session.completed":      "completed",
	"customer.subscription.updated":   "updated",
	"customer.subscription.deleted":   "cancelled",
	"invoice.payment_succeeded":       "succeeded",
	"invoice.payment_failed":          "failed",
}

func billingDisplayName(eventType string) string {
	if name, ok := billingEventDisplayNames[eventType]; ok {
		return name
	}
	return eventType
}

func billingStatus(eventType string) string {
	if status, ok := billingEventStatuses[eventType]; ok {
		return status
	}
	return eventType
}

// GetBillingHistory returns the user's billing event history
func (s *SubscriptionService) GetBillingHistory(ctx context.Context, userID string) (*dto.BillingHistoryResponse, error) {
	events, err := s.billingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	entries := make([]dto.BillingHistoryEntry, 0, len(events))
	for _, e := range events {
		amount := ""
		if e.AmountCents != nil && e.Currency != nil {
			amount = fmt.Sprintf("%.2f %s", float64(*e.AmountCents)/100.0, *e.Currency)
		}
		invoiceURL := ""
		if e.InvoiceURL != nil {
			invoiceURL = *e.InvoiceURL
		}
		entries = append(entries, dto.BillingHistoryEntry{
			ID:          e.ID,
			Date:        e.CreatedAt.Format("2006-01-02"),
			Description: billingDisplayName(e.EventType),
			Amount:      amount,
			Status:      billingStatus(e.EventType),
			InvoiceURL:  invoiceURL,
		})
	}

	return &dto.BillingHistoryResponse{Data: entries}, nil
}

// HandleWebhook processes a Stripe webhook event
func (s *SubscriptionService) HandleWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	event, err := webhook.ConstructEventWithOptions(payload, sigHeader, s.config.WebhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return fmt.Errorf("webhook signature verification failed: %w", err)
	}

	log := logger.Log

	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(ctx, event)
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(ctx, event)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(ctx, event)
	case "invoice.payment_succeeded":
		return s.handleInvoicePaymentSucceeded(ctx, event)
	case "invoice.payment_failed":
		return s.handleInvoicePaymentFailed(ctx, event)
	default:
		log.Debug("unhandled stripe event type", "type", event.Type)
	}

	return nil
}

func (s *SubscriptionService) handleCheckoutCompleted(ctx context.Context, event stripe.Event) error {
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
		return fmt.Errorf("failed to parse checkout session: %w", err)
	}

	if sess.Subscription == nil || sess.Customer == nil {
		return nil
	}

	userID := sess.Metadata["user_id"]
	if userID == "" {
		// Try to find the user by customer ID
		profile, err := s.profileRepo.GetByStripeCustomerID(ctx, sess.Customer.ID)
		if err != nil {
			return fmt.Errorf("could not find user for customer %s: %w", sess.Customer.ID, err)
		}
		userID = profile.ID
	}

	profile, err := s.profileRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	profile.IsPremium = true
	profile.StripeCustomerID = &sess.Customer.ID
	profile.StripeSubscriptionID = &sess.Subscription.ID
	profile.SubscriptionStatus = "active"
	profile.CancelAtPeriodEnd = false
	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return err
	}
	_ = s.invalidator.InvalidateProfile(ctx, userID)
	_ = s.invalidator.InvalidateProfileDTO(ctx, userID)

	return nil
}

func (s *SubscriptionService) handleSubscriptionUpdated(ctx context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("failed to parse subscription: %w", err)
	}

	profile, err := s.profileRepo.GetByStripeSubscriptionID(ctx, sub.ID)
	if err != nil {
		return fmt.Errorf("could not find user for subscription %s: %w", sub.ID, err)
	}

	profile.SubscriptionStatus = string(sub.Status)
	profile.CancelAtPeriodEnd = sub.CancelAtPeriodEnd
	if sub.CurrentPeriodEnd > 0 {
		t := time.Unix(sub.CurrentPeriodEnd, 0)
		profile.SubscriptionCurrentPeriodEnd = &t
	}
	// Keep premium active as long as subscription is active or trialing
	profile.IsPremium = sub.Status == stripe.SubscriptionStatusActive || sub.Status == stripe.SubscriptionStatusTrialing

	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return err
	}
	_ = s.invalidator.InvalidateProfile(ctx, profile.ID)
	_ = s.invalidator.InvalidateProfileDTO(ctx, profile.ID)

	return nil
}

func (s *SubscriptionService) handleSubscriptionDeleted(ctx context.Context, event stripe.Event) error {
	log := logger.FromContext(ctx)

	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("failed to parse subscription: %w", err)
	}

	profile, err := s.profileRepo.GetByStripeSubscriptionID(ctx, sub.ID)
	if err != nil {
		return fmt.Errorf("could not find user for subscription %s: %w", sub.ID, err)
	}

	// Remove premium status and profile flair
	profile.IsPremium = false
	profile.SubscriptionStatus = "cancelled"
	profile.CancelAtPeriodEnd = false
	profile.ProfileFlair = nil
	profile.UsernameColor = nil
	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return err
	}
	_ = s.invalidator.InvalidateProfile(ctx, profile.ID)
	_ = s.invalidator.InvalidateProfileDTO(ctx, profile.ID)

	// Best-effort cleanup: delete all wishlist items
	if deletedCount, err := s.wishlistRepo.DeleteAllByUserID(ctx, profile.ID); err != nil {
		log.Error("failed to delete wishlist items on subscription cancellation",
			"error", err.Error(),
			"user_id", profile.ID,
		)
	} else if deletedCount > 0 {
		log.Info("deleted wishlist items on subscription cancellation",
			"user_id", profile.ID,
			"deleted_count", deletedCount,
		)
	}

	// Best-effort cleanup: cancel excess listings (keep only 3 most recent)
	if cancelledCount, err := s.listingRepo.CancelOldestActiveListings(ctx, profile.ID, 3); err != nil {
		log.Error("failed to cancel excess listings on subscription cancellation",
			"error", err.Error(),
			"user_id", profile.ID,
		)
	} else if cancelledCount > 0 {
		log.Info("cancelled excess listings on subscription cancellation",
			"user_id", profile.ID,
			"cancelled_count", cancelledCount,
		)
	}

	return nil
}

func (s *SubscriptionService) handleInvoicePaymentSucceeded(ctx context.Context, event stripe.Event) error {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return fmt.Errorf("failed to parse invoice: %w", err)
	}

	// Deduplicate
	exists, err := s.billingRepo.ExistsByStripeEventID(ctx, event.ID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// Find user by customer ID
	if inv.Customer == nil {
		return nil
	}
	profile, err := s.profileRepo.GetByStripeCustomerID(ctx, inv.Customer.ID)
	if err != nil {
		return fmt.Errorf("could not find user for customer %s: %w", inv.Customer.ID, err)
	}

	amountCents := int(inv.AmountPaid)
	currency := string(inv.Currency)
	invoiceURL := inv.HostedInvoiceURL

	billingEvent := &models.BillingEvent{
		UserID:        profile.ID,
		StripeEventID: event.ID,
		EventType:     string(event.Type),
		AmountCents:   &amountCents,
		Currency:      &currency,
	}
	if invoiceURL != "" {
		billingEvent.InvoiceURL = &invoiceURL
	}

	if err := s.billingRepo.Create(ctx, billingEvent); err != nil {
		return err
	}

	// Ensure premium is active
	if !profile.IsPremium {
		profile.IsPremium = true
		profile.SubscriptionStatus = "active"
		if err := s.profileRepo.Update(ctx, profile); err != nil {
			return err
		}
		_ = s.invalidator.InvalidateProfile(ctx, profile.ID)
		_ = s.invalidator.InvalidateProfileDTO(ctx, profile.ID)
	}

	return nil
}

func (s *SubscriptionService) handleInvoicePaymentFailed(ctx context.Context, event stripe.Event) error {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return fmt.Errorf("failed to parse invoice: %w", err)
	}

	// Deduplicate
	exists, err := s.billingRepo.ExistsByStripeEventID(ctx, event.ID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if inv.Customer == nil {
		return nil
	}
	profile, err := s.profileRepo.GetByStripeCustomerID(ctx, inv.Customer.ID)
	if err != nil {
		return fmt.Errorf("could not find user for customer %s: %w", inv.Customer.ID, err)
	}

	amountCents := int(inv.AmountDue)
	currency := string(inv.Currency)

	billingEvent := &models.BillingEvent{
		UserID:        profile.ID,
		StripeEventID: event.ID,
		EventType:     string(event.Type),
		AmountCents:   &amountCents,
		Currency:      &currency,
	}

	if err := s.billingRepo.Create(ctx, billingEvent); err != nil {
		return err
	}

	profile.SubscriptionStatus = "past_due"
	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return err
	}
	_ = s.invalidator.InvalidateProfile(ctx, profile.ID)
	_ = s.invalidator.InvalidateProfileDTO(ctx, profile.ID)

	return nil
}

// UpdateFlair updates the user's profile flair (premium only)
func (s *SubscriptionService) UpdateFlair(ctx context.Context, userID string, flair string) error {
	profile, err := s.profileRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !profile.IsPremium {
		return ErrForbidden
	}

	if flair == "none" {
		profile.ProfileFlair = nil
	} else {
		profile.ProfileFlair = &flair
	}

	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return err
	}
	_ = s.invalidator.InvalidateProfile(ctx, userID)
	_ = s.invalidator.InvalidateProfileDTO(ctx, userID)

	return nil
}

// UpdateUsernameColor updates the user's username color (premium only)
func (s *SubscriptionService) UpdateUsernameColor(ctx context.Context, userID string, color string) error {
	profile, err := s.profileRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !profile.IsPremium {
		return ErrForbidden
	}

	if color == "none" {
		profile.UsernameColor = nil
	} else {
		profile.UsernameColor = &color
	}

	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return err
	}
	_ = s.invalidator.InvalidateProfile(ctx, userID)
	_ = s.invalidator.InvalidateProfileDTO(ctx, userID)

	return nil
}

// GetPriceHistory returns trade volume history for an item (premium only)
func (s *SubscriptionService) GetPriceHistory(ctx context.Context, userID string, itemName string, days int) (*dto.TradeVolumeResponse, error) {
	profile, err := s.profileRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !profile.IsPremium {
		return nil, ErrForbidden
	}

	if days <= 0 || days > 90 {
		days = 30
	}

	volumes, err := s.transactionRepo.GetTradeVolume(ctx, itemName, days)
	if err != nil {
		return nil, err
	}

	data := make([]dto.TradeVolumePoint, 0, len(volumes))
	for _, v := range volumes {
		data = append(data, dto.TradeVolumePoint{
			Date:   v.Date,
			Volume: v.Volume,
		})
	}

	return &dto.TradeVolumeResponse{Data: data}, nil
}

// ReadBody is a helper for reading the raw body from a request reader
func ReadBody(body io.Reader) ([]byte, error) {
	return io.ReadAll(body)
}
