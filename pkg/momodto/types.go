package momodto

type WebhookPayload struct {
	TransactionID string `json:"transactionId"`
	ExternalId    string `json:"externalId"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
	Payer         Payer  `json:"payer"`
	Timestamp     string `json:"timestamp"`
}

type Payer struct {
	PartyIDType string `json:"partyIdType"`
	PartyID     string `json:"partyId"`
}

type NotifyPayload struct {
	ExternalID string `json:"external_id"`
	Status     string `json:"status"`
	Amount     string `json:"amount"`
	Currency   string `json:"currency"`
	Paid       bool   `json:"paid"`
}
