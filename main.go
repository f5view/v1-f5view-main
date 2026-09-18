package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"marketplace/storage"
	"marketplace/validator"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// для проверки что назвние в кавычках
var addCmdRegex = regexp.MustCompile(`^add\s+"([^"]*)"\s+(\S+)\s+(\S+)$`)

func parseAddCommand(line string) (title string, price float64, stock int, err error) {

	parts := addCmdRegex.FindStringSubmatch(line)

	if len(parts) != 4 {
		return "", 0, 0, errors.New("invalid arguments")
	}

	title = parts[1]

	// Парсим цену
	price, err = strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return "", 0, 0, errors.New("invalid arguments")
	}

	// Парсим остаток
	stock, err = strconv.Atoi(parts[3])
	if err != nil {
		return "", 0, 0, errors.New("invalid arguments")
	}

	return title, price, stock, nil
}

// функция для печати ошибок в 1 формате
func PrintError(msg string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
}

func main() {
	// 1. Объявляем флаги
	// flag.String(имя, значение_по_умолчанию, описание)
	storageType := flag.String("storage", "memory", "тип хранилища: memory или json")
	filePath := flag.String("file", "", "путь к JSON-файлу")

	// 2. Читаем переданные аргументы командной строки
	flag.Parse()

	// 3. Создаем интерфейс хранилища
	var stor storage.Storage

	// 4. Логика выбора хранилища
	switch *storageType {
	case "memory":
		stor = storage.NewMemoryStorage()

	case "json":
		// Требование: при --storage=json флаг --file обязателен!
		if *filePath == "" {
			fmt.Fprintln(os.Stderr, "Error: --file flag is required when using --storage=json")
			return
		}

		var err error
		stor, err = storage.NewJSONStorage(*filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
			return
		}

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown storage type '%s'\n", *storageType)
		return
	}
	// сканер, который подключен к стандартному вводу(к консоли)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		// проверка на чтение из консоли
		if !scanner.Scan() {
			break
		}
		// записываем в переменную текст который мы ввели
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}
		if line == "exit" {
			fmt.Println("Goodbye!")
			break
		}
		// разделяем на части, если есть пробел значит 2 разных части
		parts := strings.Fields(line)
		cmd := parts[0]

		switch cmd {
		case "add":
			title, price, stock, err := parseAddCommand(line)
			if err != nil {
				PrintError(err.Error())
				continue
			}
			if err := validator.CheckValiation(title, price, stock); err != nil {
				PrintError(err.Error())
				continue
			}
			item, err := stor.Add(title, price, stock)
			if err != nil {
				PrintError(err.Error())
				continue
			}
			fmt.Printf("Listing added with ID: %d\n", item.Id)

		case "list":
			items, err := stor.List()
			if err != nil {
				PrintError(err.Error())
				continue
			}
			for _, item := range items {
				fmt.Printf("%d. \"%s\" — %.2f (stock: %d)\n", item.Id, item.Title, item.Price, item.Stock)
			}

		case "buy":
			if len(parts) != 3 {
				PrintError("invalid arguments")
				continue
			}

			// Парсим id
			id, err := strconv.Atoi(parts[1])
			if err != nil {
				PrintError("invalid arguments")
				continue
			}

			// Парсим qty
			qty, err := strconv.Atoi(parts[2])
			if err != nil {
				PrintError("invalid arguments")
				continue
			}
			stock, err := stor.Buy(id, qty)
			if err != nil {
				PrintError(err.Error())
				continue
			}
			fmt.Printf("Bought %d of listing %d (remaining: %d)\n", qty, id, stock)

		case "delete":
			if len(parts) != 2 {
				PrintError("invalid arguments")
				continue
			}

			// Парсим id
			id, err := strconv.Atoi(parts[1])
			if err != nil {
				PrintError("invalid arguments")
				continue
			}

			if err := stor.Delete(id); err != nil {
				PrintError(err.Error())
				continue
			}
			fmt.Printf("Listing %d deleted\n", id)

		default:
			PrintError(fmt.Sprintf("unknown command '%s'", cmd))
		}

	}
	// на случай каких то критических ошибок
	if err := scanner.Err(); err != nil {
		PrintError(err.Error())
	}
}
