package numRating

import (
	"strconv"
	"strings"
	"tg_market/internal/domain/entity"
)

// CalculateValue оценивает красоту числа
func CalculateValue(num int) entity.Rating {
	s := strconv.Itoa(num)
	length := len(s)

	// 1. Однозначные (1-9) -> 100%
	if num > 0 && num < 10 {
		return entity.Rating{Score: 100, Description: "Single Digit", IsUnique: true}
	}

	// 2. Одинаковые цифры (Solid) - 11111, 777 -> 100%
	if isSolid(s) {
		return entity.Rating{Score: 100, Description: "Solid", IsUnique: true}
	}

	// 3. Двузначные (10-99) -> 90%
	if num < 100 {
		return entity.Rating{90, "Double Digit", true}
	}

	// 4. Последовательности (Ladder) - 12345, 54321 -> 85%
	if isLadder(s) {
		return entity.Rating{85, "Ladder", true}
	}

	// 5. Круглые числа (Million/Kilo) - 1000, 500000 -> 80%
	if strings.HasSuffix(s, "000") && isSolid(s[:length-3]) {
		zeros := countTrailingZeros(s)
		score := 50.0 + (float64(zeros) * 10)
		if score > 95 {
			score = 95
		}
		return entity.Rating{score, "Round", true}
	}

	// 6. Трехзначные (100-999) -> 75%
	if num < 1000 {
		return entity.Rating{75, "Triple Digit", true}
	}

	// 7. Повторы (Repeater XYXY) - 1212, 6969, 123123 -> 70%
	if isRepeater(s) {
		return entity.Rating{70, "Repeater", true}
	}

	// 8. Палиндромы (Radar) - 12321, 1221 -> 65%
	if isPalindrome(s) {
		return entity.Rating{65, "Palindrome", true}
	}

	// 9. Сэндвич (XYYX внутри) или Радар 4-знак (1001) -> 40%
	if length == 4 && s[0] == s[3] && s[1] == s[2] {
		return entity.Rating{40, "Sandwich", true}
	}

	// 10. Красивые окончания (Suffix) - ...777, ...888 -> 20-30%
	if length >= 5 && isSolid(s[length-3:]) {
		return entity.Rating{25, "Lucky Suffix", true}
	}

	memeList := []int{
		67,
		52,
		69,
		420,
		666,
		777,
		1337,
		1488, // Опционально, зависит от контекста
		228,
	}

	for _, memeNum := range memeList {
		if num == memeNum {
			return entity.Rating{100, "Meme", true}
		}
	}

	// Обычный номер
	return entity.Rating{0, "Random", false}
}

func isSolid(s string) bool {
	if len(s) == 0 {
		return false
	}
	first := s[0]
	for i := 1; i < len(s); i++ {
		if s[i] != first {
			return false
		}
	}
	return true
}

func isLadder(s string) bool {
	if len(s) < 3 {
		return false
	} // 12 - не лестница

	ascending := true
	descending := true

	for i := 1; i < len(s); i++ {
		curr := int(s[i] - '0')
		prev := int(s[i-1] - '0')

		if curr != prev+1 {
			ascending = false
		}
		if curr != prev-1 {
			descending = false
		}
	}
	return ascending || descending
}

func isPalindrome(s string) bool {
	n := len(s)
	for i := 0; i < n/2; i++ {
		if s[i] != s[n-1-i] {
			return false
		}
	}
	return true
}

func isRepeater(s string) bool {
	n := len(s)
	// XYXY (4), XYZXYZ (6)
	if n%2 == 0 {
		half := n / 2
		if s[:half] == s[half:] {
			return true
		}
	}
	// XYYX (1221) - это палиндром, уже учтено
	// XYXYXY (6) - проверяем тройные повторы (121212)
	if n%3 == 0 {
		part := n / 3
		if s[:part] == s[part:2*part] && s[:part] == s[2*part:] {
			return true
		}
	}
	return false
}

func countTrailingZeros(s string) int {
	count := 0
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '0' {
			count++
		} else {
			break
		}
	}
	return count
}
