package models

import "time"

type Master struct {
	ID                    int
	Name                  string
	Address               string
	ClientBotToken        string
	AdminBotToken         string
	ClientBotUsername     string
	AdminBotUsername      string
	WelcomeText           string
	IsActive              bool
	MasterTelegramID      int64
	TrialStartedAt        *time.Time
	TrialEndsAt           *time.Time
	PaidUntil             *time.Time
	SlotIntervalMin       int
	MinHoursBeforeBooking int
	CancelLimitHours      int
	Schedule              WeekSchedule
	CreatedAt             time.Time
	Longitude             float64
	Latitude              float64
	PoiID                 string
	// Prepayment
	PrepaymentEnabled bool
	PrepaymentAmount  int
	PrepaymentDetails string
}

type DaySchedule struct {
	Start *time.Time
	End   *time.Time
}

type WeekSchedule struct {
	Mon DaySchedule
	Tue DaySchedule
	Wed DaySchedule
	Thu DaySchedule
	Fri DaySchedule
	Sat DaySchedule
	Sun DaySchedule
}

type ServiceCategory struct {
	ID        int
	MasterID  int
	Name      string
	SortOrder int
}

type Service struct {
	ID          int
	MasterID    int
	CategoryID  *int
	Name        string
	Price       int
	PriceFrom   bool
	DurationMin int
	IsActive    bool
	SortOrder   int
}

type Client struct {
	ID               int
	MasterID         int
	TelegramID       int64
	TelegramUsername string
	Name             string
	Phone            string
	ConsentGiven     bool
	ConsentGivenAt   *time.Time
	NoBroadcast      bool
	IsBlocked        bool
	CreatedAt        time.Time
	VisitCount       int
	LastVisitAt      *time.Time
}

type Booking struct {
	ID               int
	MasterID         int
	ClientID         int
	ServiceID        int
	StartsAt         time.Time
	EndsAt           time.Time
	Status           string
	ConfirmedBy      string
	CancelReason     string
	Reminder24hSent  bool
	Reminder2hSent   bool
	ReviewRequested  bool
	CreatedAt        time.Time
	PrepaymentStatus string
	// Joined fields
	ClientName         string
	ClientPhone        string
	ClientTelegramID   int64
	ServiceName        string
	ServicePrice       int
	ServiceDurationMin int
}

type BlockedSlot struct {
	ID        int
	MasterID  int
	StartsAt  time.Time
	EndsAt    time.Time
	Reason    string
	CreatedAt time.Time
}

type Review struct {
	ID        int
	MasterID  int
	ClientID  int
	BookingID int
	Text      string
	CreatedAt time.Time
	// Joined
	ClientName  string
	ServiceName string
}

// Booking statuses
const (
	StatusPending            = "pending"
	StatusAwaitingPrepayment = "awaiting_prepayment"
	StatusConfirmed          = "confirmed"
	StatusCancelledByClient  = "cancelled_by_client"
	StatusCancelledByMaster  = "cancelled_by_master"
	StatusCompleted          = "completed"
	StatusExpired            = "expired"
)

func IsActiveStatus(status string) bool {
	switch status {
	case StatusPending, StatusConfirmed:
		return true
	default:
		return false
	}
}

// User session state for multi-step flows
type PendingService struct {
	Name        string
	Price       int
	DurationMin int
}

type SessionState struct {
	Step            string
	ServiceID       int
	Date            string // "2006-01-02"
	BookingID       int    // for cancellation/reschedule
	PendingService  PendingService
	BroadcastText   string
	BroadcastMonths int
	// Prepayment setup
	PrepaymentAmount  int
	PrepaymentDetails string
}

// Steps for client bot
const (
	StepIdle                   = ""
	StepAwaitName              = "await_name"
	StepAwaitPhone             = "await_phone"
	StepSelectService          = "select_service"
	StepSelectCategory         = "select_category"
	StepSelectDate             = "select_date"
	StepSelectTime             = "select_time"
	StepConfirmBooking         = "confirm_booking"
	StepAwaitReview            = "await_review"
	StepAwaitPrepaymentAmount  = "await_prepayment_amount"
	StepAwaitPrepaymentDetails = "await_prepayment_details"
)

// Admin bot steps
const (
	StepAwaitBroadcastMsg = "await_broadcast_msg"

	// Add service
	StepAwaitServiceName     = "await_service_name"
	StepAwaitServicePrice    = "await_service_price"
	StepAwaitServiceDuration = "await_service_duration"

	// Edit service
	StepAwaitEditServiceName     = "await_edit_svc_name"
	StepAwaitEditServicePrice    = "await_edit_svc_price"
	StepAwaitEditServiceDuration = "await_edit_svc_duration"
)
