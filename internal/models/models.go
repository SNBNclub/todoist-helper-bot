package models

type User struct {
	ChatID int64
	Name   string
}

type WebHookRequest struct {
	EventName      string      `json:"event_name"`
	UserID         string      `json:"user_id"`
	EventData      interface{} `json:"event_data"` // Use `interface{}` if the structure of event_data is dynamic
	Version        string      `json:"version"`
	Initiator      interface{} `json:"initiator"` // Use `interface{}` if the structure of initiator is dynamic
	TriggeredAt    string      `json:"triggered_at"`
	EventDataExtra interface{} `json:"event_data_extra"` // Use `interface{}` if the structure of event_data_extra is dynamic
}

type Duration struct {
	Amount int    `json:"amount"`
	Unit   string `json:"unit"`
}

type Task struct {
	ID             string      `json:"id"`
	UserID         string      `json:"user_id"`
	ProjectID      string      `json:"project_id"`
	Content        string      `json:"content"`
	Description    string      `json:"description"`
	Priority       int         `json:"priority"`
	Due            interface{} `json:"due"`       // Use `interface{}` if the structure is dynamic
	Deadline       interface{} `json:"deadline"`  // Use `interface{}` if the structure is dynamic
	ParentID       *string     `json:"parent_id"` // Use pointer for nullable fields
	ChildOrder     int         `json:"child_order"`
	SectionID      *string     `json:"section_id"` // Use pointer for nullable fields
	DayOrder       int         `json:"day_order"`
	Collapsed      bool        `json:"collapsed"`
	Labels         []string    `json:"labels"`
	AddedByUID     string      `json:"added_by_uid"`
	AssignedByUID  string      `json:"assigned_by_uid"`
	ResponsibleUID *string     `json:"responsible_uid"` // Use pointer for nullable fields
	Checked        bool        `json:"checked"`
	IsDeleted      bool        `json:"is_deleted"`
	SyncID         *string     `json:"sync_id"`      // Use pointer for nullable fields
	CompletedAt    *string     `json:"completed_at"` // Use pointer for nullable fields
	AddedAt        string      `json:"added_at"`
	Duration       *Duration   `json:"duration"` // Use pointer for nested objects
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
