package value

type GiftAttributes struct {
	Model          string `json:"model,omitempty"`
	Backdrop       string `json:"backdrop,omitempty"`
	Symbol         string `json:"symbol,omitempty"`
	Pattern        string `json:"pattern,omitempty"`
	RarityPerMille int    `json:"rarity,omitempty"`
}
