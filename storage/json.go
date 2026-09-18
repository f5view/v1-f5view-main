package storage

import (
	"encoding/json"
	"os"
)

// fileData — структура для сериализации в JSON-файл иоже самое что и в памяти только добавляем еще тэги
type fileData struct {
	NextID int       `json:"next_id"`
	Items  []Listing `json:"items"`
}

// JSONStorage хранит данные в файле на диске тож самое как с хранением в памяти но добавляем еще путь
type JSONStorage struct {
	filePath string
	items    []Listing
	nextID   int
}

// NewJSONStorage загружает существующий файл или создает новое пустое хранилище короче конструктор
func NewJSONStorage(filePath string) (*JSONStorage, error) {
	s := &JSONStorage{
		filePath: filePath,
		items:    make([]Listing, 0),
		nextID:   1, // по умолчанию начинаем с 1
	}

	// 1. Проверяем, существует ли файл
	data, err := os.ReadFile(filePath)
	if err != nil {
		// Если файла еще нет создаем новый
		if os.IsNotExist(err) {
			return s, nil
		}
		// Какая-то системная ошибка чтения
		return nil, err
	}

	// Если файл пустой (0 байт), ничего не парсим
	if len(data) == 0 {
		return s, nil
	}

	// 2. Десериализуем JSON в структуру
	var fd fileData
	if err := json.Unmarshal(data, &fd); err != nil { // 2 аргумент не забываем что указатель
		return nil, err
	}

	// 3. Заполняем состояние
	s.items = fd.Items
	if fd.NextID > 0 {
		s.nextID = fd.NextID
	}

	return s, nil
}

// save — приватный метод, который сбрасывает текущее состояние в файл
func (s *JSONStorage) save() error {
	fd := fileData{
		NextID: s.nextID,
		Items:  s.items,
	}

	// Превращаем структуру в красивые байты JSON с отступами в 2 пробела
	bytes, err := json.MarshalIndent(fd, "", "  ")
	if err != nil {
		return err
	}

	// Записываем в файл (0644 — стандартные права доступа на чтение/запись)
	return os.WriteFile(s.filePath, bytes, 0644)
}

// Add добавляет товар и сохраняет в JSON такие же методы как и в памяти не зря у нас интерфейс
func (s *JSONStorage) Add(title string, price float64, stock int) (Listing, error) {
	item := Listing{
		Id:    s.nextID,
		Title: title,
		Price: price,
		Stock: stock,
	}

	s.items = append(s.items, item)
	s.nextID++

	// Сбрасываем изменения в файл!
	if err := s.save(); err != nil {
		return Listing{}, err
	}

	return item, nil
}

// List просто возвращает текущие элементы
func (s *JSONStorage) List() ([]Listing, error) {
	return s.items, nil
}

// Delete удаляет товар и обновляет файл
func (s *JSONStorage) Delete(id int) error {
	found := false
	for i, item := range s.items {
		if item.Id == id {
			s.items = append(s.items[:i], s.items[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return ErrIdNotFound
	}

	return s.save()
}

// Buy уменьшает остаток и обновляет файл
func (s *JSONStorage) Buy(id int, qty int) (int, error) {
	if qty <= 0 {
		return 0, Errqty
	}

	for i := range s.items {
		if s.items[i].Id == id {
			if s.items[i].Stock < qty {
				return 0, ErrqtyMoreStock
			}

			s.items[i].Stock -= qty

			// Сохраняем изменение на диск
			if err := s.save(); err != nil {
				return 0, err
			}

			return s.items[i].Stock, nil
		}
	}

	return 0, ErrIdNotFound
}
