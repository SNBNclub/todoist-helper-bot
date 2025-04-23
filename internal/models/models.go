package models

import "time"

type TgUser struct {
	ChatID int64
	Name   string
}

type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type WebHookRequest struct {
	EventName      string      `json:"event_name"`
	UserID         string      `json:"user_id"`
	EventData      interface{} `json:"event_data"` // Use `interface{}` if the structure of event_data is dynamic
	Version        string      `json:"version"`
	Initiator      Initiator   `json:"initiator"`
	TriggeredAt    string      `json:"triggered_at"`
	EventDataExtra interface{} `json:"event_data_extra"` // Use `interface{}` if the structure of event_data_extra is dynamic
}

type Task struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	ProjectID      string    `json:"project_id"`
	Content        string    `json:"content"`
	Description    string    `json:"description"`
	Priority       int       `json:"priority"`
	Due            any       `json:"due"`
	Deadline       any       `json:"deadline"`
	ParentID       any       `json:"parent_id"`
	ChildOrder     int       `json:"child_order"`
	SectionID      string    `json:"section_id"`
	DayOrder       int       `json:"day_order"`
	Collapsed      bool      `json:"collapsed"`
	Labels         []string  `json:"labels"`
	AddedByUID     string    `json:"added_by_uid"`
	AssignedByUID  string    `json:"assigned_by_uid"`
	ResponsibleUID any       `json:"responsible_uid"`
	Checked        bool      `json:"checked"`
	IsDeleted      bool      `json:"is_deleted"`
	AddedAt        time.Time `json:"added_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CompletedAt    any       `json:"completed_at"`
	// TODO :: is pointer to structure correct in json umnmarshaling
	Duration *Duration `json:"duration"`
}

// TODO :: rename
type WebHookParsed struct {
	UserID    string
	TimeSpent uint32
	AskTime   bool
}

type Initiator struct {
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	ID        string `json:"id"`
	ImageID   string `json:"image_id"`
	IsPremium bool   `json:"is_premium"`
}

type Duration struct {
	Amount int    `json:"amount"`
	Unit   string `json:"unit"`
}

// probably unneeded
type UpdateItemRequest struct {
	ID             string                 `json:"id"`
	Content        string                 `json:"content"`
	Description    string                 `json:"description"`
	Due            map[string]interface{} `json:"due"`      // Use map[string]interface{} if the structure is dynamic
	Deadline       map[string]interface{} `json:"deadline"` // Use map[string]interface{} if the structure is dynamic
	Priority       int                    `json:"priority"`
	Collapsed      bool                   `json:"collapsed"`
	Labels         []string               `json:"labels"`
	AssignedByUID  string                 `json:"assigned_by_uid"`
	ResponsibleUID string                 `json:"responsible_uid"`
	DayOrder       int                    `json:"day_order"`
	Duration       map[string]interface{} `json:"duration"` // Use map[string]interface{} if the structure is dynamic
}

type InitSyncReq struct {
	FullSync      bool        `json:"full_sync"`
	SyncToken     string      `json:"sync_token"`
	TempIDMapping interface{} `json:"temp_id_mapping"`
	User          SyncUser    `json:"user"`
}

type SyncUser struct {
	ActivatedUser     bool   `json:"activated_user"`
	AutoReminder      int    `json:"auto_reminder"`
	AvatarBig         string `json:"avatar_big"`
	AvatarMedium      string `json:"avatar_medium"`
	AvatarS640        string `json:"avatar_s640"`
	AvatarSmall       string `json:"avatar_small"`
	BusinessAccountID string `json:"business_account_id"`
	DailyGoal         int    `json:"daily_goal"`
	DateFormat        int    `json:"date_format"`
	DaysOff           []int  `json:"days_off"`
	Email             string `json:"email"`
	FeatureIdentifier string `json:"feature_identifier"`
	Features          struct {
		Beta                  int  `json:"beta"`
		DateistInlineDisabled bool `json:"dateist_inline_disabled"`
		DateistLang           any  `json:"dateist_lang"`
		GlobalTeams           bool `json:"global.teams"`
		HasPushReminders      bool `json:"has_push_reminders"`
		KarmaDisabled         bool `json:"karma_disabled"`
		KarmaVacation         bool `json:"karma_vacation"`
		KisaConsentTimestamp  any  `json:"kisa_consent_timestamp"`
		Restriction           int  `json:"restriction"`
	} `json:"features"`
	FullName              string    `json:"full_name"`
	HasPassword           bool      `json:"has_password"`
	ID                    string    `json:"id"`
	ImageID               string    `json:"image_id"`
	InboxProjectID        string    `json:"inbox_project_id"`
	IsCelebrationsEnabled bool      `json:"is_celebrations_enabled"`
	IsPremium             bool      `json:"is_premium"`
	JoinableWorkspace     any       `json:"joinable_workspace"`
	JoinedAt              time.Time `json:"joined_at"`
	Karma                 int       `json:"karma"`
	KarmaTrend            string    `json:"karma_trend"`
	Lang                  string    `json:"lang"`
	MfaEnabled            bool      `json:"mfa_enabled"`
	NextWeek              int       `json:"next_week"`
	PremiumStatus         string    `json:"premium_status"`
	PremiumUntil          any       `json:"premium_until"`
	ShareLimit            int       `json:"share_limit"`
	SortOrder             int       `json:"sort_order"`
	StartDay              int       `json:"start_day"`
	StartPage             string    `json:"start_page"`
	ThemeID               string    `json:"theme_id"`
	TimeFormat            int       `json:"time_format"`
	Token                 string    `json:"token"`
	TzInfo                struct {
		GmtString string `json:"gmt_string"`
		Hours     int    `json:"hours"`
		IsDst     int    `json:"is_dst"`
		Minutes   int    `json:"minutes"`
		Timezone  string `json:"timezone"`
	} `json:"tz_info"`
	VerificationStatus string `json:"verification_status"`
	WeekendStartDay    int    `json:"weekend_start_day"`
	WeeklyGoal         int    `json:"weekly_goal"`
}
