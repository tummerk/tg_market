package main

import (
    "fmt"
    "C:\\Users\\никита\\Desktop\\tg_market\\pkg\\numrating"
)

func main() {
    // Создаем новый сервис
    service := numrating.New()
    
    // Заполняем массив мемных чисел (ваша задача - заполнить этот массив)
    memeNumbers := []int{69, 420, 1337, 80085, 8008135, 666, 777}
    service.SetMemeNumbers(memeNumbers)
    
    // Пример сравнения чисел с мемными
    testNumbers := []int{69, 123, 420, 999}
    
    for _, num := range testNumbers {
        results := service.Compare(num)
        fmt.Printf("Число %d сравнивается с мемными: %v\n", num, results)
        
        // Показываем, с какими именно мемными числами совпадает
        matches := []int{}
        for i, match := range results {
            if match {
                matches = append(matches, service.GetMemeNumbers()[i])
            }
        }
        if len(matches) > 0 {
            fmt.Printf("  Совпадает с: %v\n", matches)
        } else {
            fmt.Printf("  Не совпадает ни с одним мемным числом\n")
        }
    }
}