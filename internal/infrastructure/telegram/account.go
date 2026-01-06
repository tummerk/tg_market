package telegram

import (
	"encoding/json"
	"os"
)

type Account struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

func LoadAccounts(path string) ([]Account, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var accounts []Account
	return accounts, json.Unmarshal(data, &accounts)
}
