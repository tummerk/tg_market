// Данный файл должен быть сгенерирован из openapi спецификации и называться types.gen.go
package rest

type Example struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Error Модель ошибок
type Error struct {
	// Code Код ошибки
	Code ErrorCode `json:"code"`

	// Message Сообщение об ошибке (для отображения в UI в будущем)
	Message string `json:"message"`
}

// ErrorCode Код ошибки
type ErrorCode string
