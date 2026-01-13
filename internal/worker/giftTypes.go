package worker

// AddGiftType добавляет ID в список сканирования (если ещё нет)
func (w *MarketScanner) AddGiftType(id int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Проверяем, нет ли уже такого ID
	for _, existingID := range w.giftTypeIDs {
		if existingID == id {
			return
		}
	}

	w.giftTypeIDs = append(w.giftTypeIDs, id)
}

// AddGiftTypes добавляет несколько ID
func (w *MarketScanner) AddGiftTypes(ids ...int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, id := range ids {
		exists := false
		for _, existingID := range w.giftTypeIDs {
			if existingID == id {
				exists = true
				break
			}
		}
		if !exists {
			w.giftTypeIDs = append(w.giftTypeIDs, id)
		}
	}
}

// RemoveGiftType удаляет ID из списка сканирования
func (w *MarketScanner) RemoveGiftType(id int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i, existingID := range w.giftTypeIDs {
		if existingID == id {
			// Удаляем элемент, сохраняя порядок
			w.giftTypeIDs = append(w.giftTypeIDs[:i], w.giftTypeIDs[i+1:]...)
			return
		}
	}
}

// GetGiftTypes возвращает копию текущего списка ID
func (w *MarketScanner) GetGiftTypes() []int64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.giftTypeIDs) == 0 {
		return nil
	}

	// Возвращаем копию, чтобы избежать race condition
	result := make([]int64, len(w.giftTypeIDs))
	copy(result, w.giftTypeIDs)
	return result
}

// SetGiftTypes заменяет весь список ID
func (w *MarketScanner) SetGiftTypes(ids []int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(ids) == 0 {
		w.giftTypeIDs = nil
		return
	}

	w.giftTypeIDs = make([]int64, len(ids))
	copy(w.giftTypeIDs, ids)
}

// ClearGiftTypes очищает список (будут сканироваться все типы)
func (w *MarketScanner) ClearGiftTypes() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.giftTypeIDs = nil
}

// HasGiftType проверяет, есть ли ID в списке
func (w *MarketScanner) HasGiftType(id int64) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, existingID := range w.giftTypeIDs {
		if existingID == id {
			return true
		}
	}
	return false
}
