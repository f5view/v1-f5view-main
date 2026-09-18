package storage

import "errors"

// наши ошибочки которые мы тут будем использовать
var (
	Errqty          = errors.New("quantity must be positive")
	ErrqtyMoreStock = errors.New("insufficient stock")
	ErrIdNotFound   = errors.New("listing not found")
)

type MemoryStorage struct {
	items  []Listing // слайс структур
	nextId int       // всегда помним след айди так как можем удалить максимальный айди и данные запишуться под удаленным айдишник что плохо для нашего случая
}

func NewMemoryStorage() *MemoryStorage { // конструктор чтобы сразу присвоить айди 1 и использовать эту структуру
	return &MemoryStorage{
		items:  make([]Listing, 0),
		nextId: 1,
	}
}

func (m *MemoryStorage) Add(title string, price float64, stock int) (Listing, error) {
	item := Listing{
		Id:    m.nextId,
		Title: title,
		Price: price,
		Stock: stock,
	}

	m.items = append(m.items, item)
	m.nextId++ // увеличиваем айди

	return item, nil
}

func (m *MemoryStorage) List() ([]Listing, error) {
	return m.items, nil // просто выводим все наши предметы
}

func (m *MemoryStorage) Delete(id int) error {
	for i, val := range m.items { // проходимся по всему предметам
		if val.Id == id { // если айди предмета совпадает с запрошенным айди то это оно
			m.items = append(m.items[:i], m.items[i+1:]...) // базове удаление увидел из статьи про утечку в slice
			return nil
		}
	}
	return ErrIdNotFound
}

func (m *MemoryStorage) Buy(id int, qty int) (int, error) {
	if qty <= 0 { // проверяем тут
		return 0, Errqty
	}
	for i, val := range m.items {
		if val.Id == id {
			if m.items[i].Stock < qty { // тяжело, но у предмета с правильным айдишником он стучиться в остаток и все гуд :)
				return 0, ErrqtyMoreStock
			}

			m.items[i].Stock -= qty // меням остаток
			return m.items[i].Stock, nil
		}
	}
	return 0, ErrIdNotFound
}
