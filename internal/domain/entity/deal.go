package entity

type Deal struct {
	// Основная информация о лоте
	Gift     *Gift
	GiftType *GiftType

	// Экономические показатели (почему мы решили купить)
	AvgPrice int64   // Текущая рыночная (AvgPrice)
	Profit   float64 // Ожидаемая прибыль (в %) или Discount

	// Технические данные для мгновенной покупки (чтобы не искать заново)
	// Эти поля можно добавить, если Gift внутри себя их не хранит
	SellerAccessHash int64 `json:"-"` // Не сериализуем в логи
}
