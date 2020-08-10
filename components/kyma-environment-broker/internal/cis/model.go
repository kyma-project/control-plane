package cis

type Event struct {
	CreationTime int64  `json:"creationTime"`
	SubAccount   string `json:"entityId"`
	Type         string `json:"eventType"`
}

type CisResponse struct {
	Total      int     `json:"total"`
	TotalPages int     `json:"totalPages"`
	PageNum    int     `json:"pageNum"`
	Events     []Event `json:"events"`
}

type EventDataVer1 struct {
	SubAccount string `json:"tenantName"`
}

type EventVer1 struct {
	Type string `json:"type"`
	Data string `json:"eventData"`
}

type CisResponseVer1 struct {
	Total      int         `json:"totalResults"`
	TotalPages int         `json:"totalPages"`
	Events     []EventVer1 `json:"events"`
}
