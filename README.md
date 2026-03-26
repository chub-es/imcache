# Высокопроизводительный кэш в оперативной памяти

Готовая к использованию высокопроизводительная, потокобезопасная реализация кэша в оперативной памяти с поддержкой обобщений.

## Функции

- **Высокая производительность**: Шардированная архитектура для минимизации конфликтов блокировок
- **Потокобезопасность**: Безопасность при одновременном доступе из нескольких горутин
- **Типобезопасность**: Полная поддержка обобщений
- **Поддержка TTL**: Индивидуальное время истечения срока действия для каждого ключа
- **Автоматическая очистка**: Фоновая очистка просроченных записей
- **Обратные вызовы**: Хуки OnEvict, OnHit, OnMiss
- **Ограничения размера**: Дополнительный максимальный размер кэша
- **Корректное завершение работы**: Надлежащая очистка и освобождение ресурсов

## Производительность

Кэш разработан для высокой производительности:

- **Шардированная конструкция**: Множественные шарды уменьшают конкуренцию за блокировки
- **Оптимизированная блокировка**: Мьютексы для чтения и записи для каждого шарда
- **Нулевое выделение памяти**: Минимизация выделений памяти в горячих путях
- **Быстрое хеширование**: Хэш FNV-1a для распределения ключей

## Установка
```bash
go get github.com/chub-es/imcache
```

## Usage
```go
package main

import (
    "fmt"
    "time"
    cache "github.com/chub-es/imcache"
)

func main() {
    // Создать новый кэш с параметрами по умолчанию
    c := cache.New[string]()
    defer c.Close()
    
    // Установите значение с параметром TTL по умолчанию
    c.Set("greeting", "Hello, World!")
    
    // Установлено пользовательское значение TTL
    c.Set("temp", "This expires", 5*time.Second)
    
    // Получить значение
    if val, err := c.Get("greeting"); err == nil {
        fmt.Println(val)
    }
}
```

## Расширенное использование
### С опциями
```go
type User struct {
    ID    int
    Name  string
    Email string
}

c := cache.New[User](
    cache.WithDefaultTTL[User](10*time.Minute),
    cache.WithCleanupInterval[User](1*time.Minute),
    cache.WithMaxSize[User](10000),
    cache.WithShards[User](64),
    cache.WithOnEvict[User](func(key string, user User) {
        log.Printf("User %s evicted", user.Name)
    }),
)
```

### Типобезопасный кэш
```go
type User struct {
    ID    int
    Name  string
    Email string
}

userCache := cache.New[User]()
userCache.Set("user:1", User{ID: 1, Name: "Petya", Email: "petya@example.com"})

if user, err := userCache.Get("user:1"); err == nil {
    fmt.Printf("Found user: %s", user.Name)
}
```
