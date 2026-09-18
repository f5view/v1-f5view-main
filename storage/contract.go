package storage

type Listing struct { // наш предмет
	Id    int     `json:"id"`
	Title string  `json:"title"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

type Storage interface { // интерфейс который должны реализовывать наши структуры
	Add(title string, price float64, stock int) (Listing, error)
	List() ([]Listing, error)
	Delete(id int) error
	Buy(id int, qty int) (int, error)
}
