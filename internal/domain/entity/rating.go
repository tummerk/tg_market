package entity

// Rating результат оценки номера
type Rating struct {
	Score       float64 // 0.0 - 100.0
	Description string  // Например: "Solid (777)", "Ladder (123)"
	IsUnique    bool    // Флаг, что номер имеет ценность
}
