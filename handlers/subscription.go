package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ridwanFatur/hukumai-project-backend/db"
	"github.com/ridwanFatur/hukumai-project-backend/models"
	"github.com/stripe/stripe-go/v82"
	stripeSession "github.com/stripe/stripe-go/v82/checkout/session"
	stripeCustomer "github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/webhook"
)

// GetPlans returns all active subscription plans (public endpoint).
func GetPlans(c *gin.Context) {
	var plans []models.SubscriptionPlan
	if err := db.DB.Where("is_active = ?", true).Order("sort_order ASC").Find(&plans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data paket"})
		return
	}

	type PlanResponse struct {
		ID        uint     `json:"id"`
		Name      string   `json:"name"`
		Slug      string   `json:"slug"`
		Price     int64    `json:"price"`
		Currency  string   `json:"currency"`
		Interval  string   `json:"interval"`
		Features  []string `json:"features"`
		IsActive  bool     `json:"is_active"`
		SortOrder int      `json:"sort_order"`
	}

	result := make([]PlanResponse, 0, len(plans))
	for _, p := range plans {
		var features []string
		json.Unmarshal([]byte(p.Features), &features) //nolint:errcheck
		if features == nil {
			features = []string{}
		}
		result = append(result, PlanResponse{
			ID:        p.ID,
			Name:      p.Name,
			Slug:      p.Slug,
			Price:     p.Price,
			Currency:  p.Currency,
			Interval:  p.Interval,
			Features:  features,
			IsActive:  p.IsActive,
			SortOrder: p.SortOrder,
		})
	}

	c.JSON(http.StatusOK, gin.H{"plans": result})
}

// GetSubscriptionStatus returns the authenticated user's active subscription.
// Falls back to the Free plan info if no active subscription exists.
func GetSubscriptionStatus(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	var sub models.Subscription
	err := db.DB.Preload("Plan").
		Where("user_id = ? AND status = ?", user.ID, "active").
		First(&sub).Error

	if err != nil {
		// No active paid subscription — return Free plan info
		var freePlan models.SubscriptionPlan
		if dbErr := db.DB.Where("slug = ?", "free").First(&freePlan).Error; dbErr != nil {
			c.JSON(http.StatusOK, gin.H{"subscription": nil})
			return
		}
		var features []string
		json.Unmarshal([]byte(freePlan.Features), &features) //nolint:errcheck
		if features == nil {
			features = []string{}
		}
		c.JSON(http.StatusOK, gin.H{
			"subscription": gin.H{
				"plan_slug":  "free",
				"plan_name":  freePlan.Name,
				"plan_price": freePlan.Price,
				"features":   features,
				"status":     "active",
			},
		})
		return
	}

	var features []string
	json.Unmarshal([]byte(sub.Plan.Features), &features) //nolint:errcheck
	if features == nil {
		features = []string{}
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription": gin.H{
			"id":                     sub.ID,
			"plan_slug":              sub.Plan.Slug,
			"plan_name":              sub.Plan.Name,
			"plan_price":             sub.Plan.Price,
			"features":               features,
			"status":                 sub.Status,
			"stripe_subscription_id": sub.StripeSubscriptionID,
			"current_period_start":   sub.CurrentPeriodStart,
			"current_period_end":     sub.CurrentPeriodEnd,
		},
	})
}

// CreateCheckout creates a Stripe Checkout session for the given plan.
// Only plans with a StripePriceID are eligible (i.e. Pro plan).
func CreateCheckout(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	var req struct {
		PlanID     uint   `json:"plan_id" binding:"required"`
		SuccessURL string `json:"success_url" binding:"required"`
		CancelURL  string `json:"cancel_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data permintaan tidak valid"})
		return
	}

	var plan models.SubscriptionPlan
	if err := db.DB.Where("id = ? AND is_active = ?", req.PlanID, true).First(&plan).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Paket tidak ditemukan"})
		return
	}

	if plan.StripePriceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Paket ini tidak mendukung pembayaran online"})
		return
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	// Reuse existing Stripe customer if available
	customerID := ""
	var existingSub models.Subscription
	if db.DB.Where("user_id = ? AND stripe_customer_id != ?", user.ID, "").
		Order("created_at DESC").First(&existingSub).Error == nil {
		customerID = existingSub.StripeCustomerID
	}

	if customerID == "" {
		cus, err := stripeCustomer.New(&stripe.CustomerParams{
			Email: stripe.String(user.Email),
			Name:  stripe.String(user.Name),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat akun pelanggan"})
			return
		}
		customerID = cus.ID
	}

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(plan.StripePriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(req.SuccessURL),
		CancelURL:  stripe.String(req.CancelURL),
	}
	params.AddMetadata("user_id", fmt.Sprintf("%d", user.ID))
	params.AddMetadata("plan_id", fmt.Sprintf("%d", plan.ID))

	sess, err := stripeSession.New(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat sesi pembayaran"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"checkout_url": sess.URL})
}

// StripeWebhook handles incoming Stripe webhook events.
// Stripe signature is verified using STRIPE_WEBHOOK_SECRET.
func StripeWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal membaca payload"})
		return
	}

	sig := c.GetHeader("Stripe-Signature")
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

	event, err := webhook.ConstructEventWithOptions(payload, sig, webhookSecret,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stripe signature tidak valid"})
		return
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	switch event.Type {
	case "checkout.session.completed":
		handleCheckoutCompleted(event)
	case "customer.subscription.created", "customer.subscription.updated":
		handleSubscriptionUpserted(event)
	case "customer.subscription.deleted":
		handleSubscriptionDeleted(event)
	case "invoice.paid":
		handleInvoicePaid(event)
	case "invoice.payment_failed":
		handleInvoicePaymentFailed(event)
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

func handleCheckoutCompleted(event stripe.Event) {
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
		return
	}
	if sess.Mode != stripe.CheckoutSessionModeSubscription {
		return
	}

	userIDStr := sess.Metadata["user_id"]
	planIDStr := sess.Metadata["plan_id"]
	if userIDStr == "" || planIDStr == "" {
		return
	}

	userIDVal, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		return
	}
	planIDVal, err := strconv.ParseUint(planIDStr, 10, 64)
	if err != nil {
		return
	}
	userID := uint(userIDVal)
	planID := uint(planIDVal)

	customerID := ""
	if sess.Customer != nil {
		customerID = sess.Customer.ID
	}
	subscriptionID := ""
	if sess.Subscription != nil {
		subscriptionID = sess.Subscription.ID
	}

	// Cancel existing active subscriptions for this user
	db.DB.Model(&models.Subscription{}).
		Where("user_id = ? AND status = ?", userID, "active").
		Update("status", "canceled")

	sub := models.Subscription{
		UserID:               userID,
		PlanID:               planID,
		Status:               "active",
		StripeCustomerID:     customerID,
		StripeSubscriptionID: subscriptionID,
		CreatedAt:            time.Now(),
	}
	db.DB.Create(&sub)
}

func handleSubscriptionUpserted(event stripe.Event) {
	var stripeSub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &stripeSub); err != nil {
		return
	}

	status := "active"
	switch stripeSub.Status {
	case stripe.SubscriptionStatusCanceled:
		status = "canceled"
	case stripe.SubscriptionStatusPastDue:
		status = "past_due"
	}

	db.DB.Model(&models.Subscription{}).
		Where("stripe_subscription_id = ?", stripeSub.ID).
		Update("status", status)
}

func handleSubscriptionDeleted(event stripe.Event) {
	var stripeSub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &stripeSub); err != nil {
		return
	}
	db.DB.Model(&models.Subscription{}).
		Where("stripe_subscription_id = ?", stripeSub.ID).
		Update("status", "canceled")
}

func handleInvoicePaid(event stripe.Event) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return
	}
	subID := extractSubscriptionIDFromInvoice(&invoice)
	if subID == "" {
		return
	}
	start := time.Unix(invoice.PeriodStart, 0)
	end := time.Unix(invoice.PeriodEnd, 0)
	db.DB.Model(&models.Subscription{}).
		Where("stripe_subscription_id = ?", subID).
		Updates(map[string]interface{}{
			"status":               "active",
			"current_period_start": &start,
			"current_period_end":   &end,
		})
}

func handleInvoicePaymentFailed(event stripe.Event) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return
	}
	subID := extractSubscriptionIDFromInvoice(&invoice)
	if subID == "" {
		return
	}
	db.DB.Model(&models.Subscription{}).
		Where("stripe_subscription_id = ?", subID).
		Update("status", "past_due")
}

// extractSubscriptionIDFromInvoice returns the Stripe subscription ID from an invoice.
// In Stripe Go v82, subscription details moved under invoice.Parent.SubscriptionDetails.
func extractSubscriptionIDFromInvoice(invoice *stripe.Invoice) string {
	if invoice.Parent != nil &&
		invoice.Parent.SubscriptionDetails != nil &&
		invoice.Parent.SubscriptionDetails.Subscription != nil {
		return invoice.Parent.SubscriptionDetails.Subscription.ID
	}
	return ""
}
