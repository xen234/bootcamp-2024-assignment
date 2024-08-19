package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/xen234/bootcamp-2024-assignment/api"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Начало транзакции
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Выполнение SQL-запросов для создания таблиц
	queries := []string{
		`CREATE TABLE IF NOT EXISTS houses (
			id INTEGER PRIMARY KEY,
			unique_id TEXT NOT NULL UNIQUE, -- Уникальный номер дома
			address TEXT NOT NULL,
			year INTEGER NOT NULL,
			developer TEXT,
			created_at DATETIME NOT NULL,
			update_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS flats (
			id INTEGER PRIMARY KEY,
			house_unique_id TEXT NOT NULL, -- Ссылка на уникальный номер дома
			flat_id INTEGER NOT NULL,
			price INTEGER NOT NULL,
			rooms INTEGER NOT NULL,
			status TEXT NOT NULL,
			FOREIGN KEY (house_unique_id) REFERENCES houses(unique_id)
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			user_type TEXT NOT NULL,
			token TEXT
		);`,
	}

	for _, query := range queries {
		_, err = tx.Exec(query)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("%s: %w", op, err)
		}
	}

	// Завершение транзакции
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) CreateHouse(house api.House) (api.House, error) {
	if s.db == nil {
		return api.House{}, fmt.Errorf("database connection is nil")
	}

	const op = "storage.sqlite.CreateHouse"

	// Проверяем, существует ли дом с таким же уникальным номером
	var existingId api.HouseId
	err := s.db.QueryRow(`
        SELECT id FROM houses WHERE unique_id = ?
    `, house.Id).Scan(&existingId)
	if err != nil && err != sql.ErrNoRows {
		return api.House{}, fmt.Errorf("%s: %w", op, err)
	}

	// Если дом уже существует, возвращаем ошибку
	if existingId != 0 {
		return api.House{}, fmt.Errorf("%s: house already exists with unique id: %d", op, house.Id)
	}

	// Если дом не существует, выполняем вставку
	stmt, err := s.db.Prepare(`
        INSERT INTO houses (unique_id, address, created_at, developer, update_at, year)
        VALUES (?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return api.House{}, fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(
		house.Id,
		house.Address,
		time.Now().Format("2006-01-02 15:04:05"),
		house.Developer,
		house.UpdateAt,
		house.Year,
	)
	if err != nil {
		return api.House{}, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	// Получаем идентификатор последней вставленной строки
	id, err := res.LastInsertId()
	if err != nil {
		return api.House{}, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	// Извлекаем созданный дом из базы данных
	row := s.db.QueryRow(`
        SELECT unique_id, address, created_at, developer, update_at, year
        FROM houses WHERE id = ?
    `, id)

	var createdHouse api.House
	err = row.Scan(
		&createdHouse.Id,
		&createdHouse.Address,
		&createdHouse.CreatedAt,
		&createdHouse.Developer,
		&createdHouse.UpdateAt,
		&createdHouse.Year,
	)
	if err != nil {
		return api.House{}, fmt.Errorf("%s: failed to scan row: %w", op, err)
	}

	return createdHouse, nil
}

func (s *Storage) UpdateHouseTimestamp(houseId api.HouseId) error {
	const op = "storage.sqlite.UpdateHouseTimestamp"

	stmt, err := s.db.Prepare("UPDATE houses SET update_at = datetime('now') WHERE unique_id = ?")
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(houseId)
	if err != nil {
		return fmt.Errorf("%s: execute statement: %w", op, err)
	}

	// Проверяем количество затронутых строк
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: failed to get rows affected: %w", op, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: house with id %d does not exist", op, houseId)
	}

	return nil
}

func (s *Storage) CreateFlat(flatToCreate api.Flat) (api.Flat, error) {
	const op = "storage.sqlite.CreateFlat"

	// Проверяем, существует ли дом с таким же уникальным номером
	var existingId api.HouseId
	err := s.db.QueryRow(`
        SELECT id FROM houses WHERE unique_id = ?
    `, flatToCreate.HouseId).Scan(&existingId)
	if err != nil && err != sql.ErrNoRows {
		return api.Flat{}, fmt.Errorf("%s: %w", op, err)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return api.Flat{}, fmt.Errorf("%s: house does not exist with unique id: %d", op, flatToCreate.HouseId)
		}
		return api.Flat{}, fmt.Errorf("%s: %w", op, err)
	}

	// Проверка существования квартиры с таким номером
	var existingFlatId api.FlatId
	err = s.db.QueryRow(`
        SELECT id FROM flats WHERE flat_id = ? AND house_unique_id = ?
    `, flatToCreate.Id, flatToCreate.HouseId).Scan(&existingFlatId)
	if err != nil && err != sql.ErrNoRows {
		return api.Flat{}, fmt.Errorf("%s: %w", op, err)
	}

	if err == nil {
		// Квартира с таким номером уже существует
		return api.Flat{}, fmt.Errorf("%s: flat with id %d already exists in house with id %d", op, flatToCreate.Id, flatToCreate.HouseId)
	}

	// Подготовка SQL-запроса для вставки новой записи в таблицу flats
	stmt, err := s.db.Prepare("INSERT INTO flats (house_unique_id, flat_id, price, rooms, status) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return api.Flat{}, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	// Выполнение SQL-запроса с параметрами
	_, err = stmt.Exec(
		flatToCreate.HouseId,
		flatToCreate.Id,
		flatToCreate.Price,
		flatToCreate.Rooms,
		api.Created,
	)

	if err != nil {
		return api.Flat{}, fmt.Errorf("%s: %w", op, err)
	}

	createdFlat := api.Flat{
		Id:      flatToCreate.Id,
		HouseId: flatToCreate.HouseId,
		Price:   flatToCreate.Price,
		Rooms:   flatToCreate.Rooms,
		Status:  api.Created,
	}

	return createdFlat, nil
}

func (s *Storage) UpdateFlat(flat api.Flat) (api.Flat, error) {
	const op = "storage.sqlite.UpdateFlat"

	// Подготовка запроса на обновление с учетом уникального идентификатора дома
	stmt, err := s.db.Prepare("UPDATE flats SET status = ? WHERE flat_id = ? AND house_unique_id = ?")
	if err != nil {
		return api.Flat{}, fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	// Выполнение запроса на обновление
	_, err = stmt.Exec(flat.Status, flat.Id, flat.HouseId)
	if err != nil {
		return api.Flat{}, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	// Подготовка запроса на извлечение обновленной записи
	row := s.db.QueryRow("SELECT house_unique_id, flat_id, price, rooms, status FROM flats WHERE flat_id = ? AND house_unique_id = ?", flat.Id, flat.HouseId)

	// Извлечение обновленных данных
	var updatedFlat api.Flat
	err = row.Scan(&updatedFlat.HouseId, &updatedFlat.Id, &updatedFlat.Price, &updatedFlat.Rooms, &updatedFlat.Status)
	if err != nil {
		return api.Flat{}, fmt.Errorf("%s: scan row: %w", op, err)
	}

	// Возврат обновленной квартиры
	return updatedFlat, nil
}

func (s *Storage) GetAllFlatsByHouseId(houseId api.HouseId) ([]api.Flat, error) {
	const op = "storage.sqlite.GetAllFlatsByHouseId"

	// Запрос на получение всех квартир по уникальному идентификатору дома
	rows, err := s.db.Query("SELECT house_unique_id, flat_id, price, rooms, status FROM flats WHERE house_unique_id = ?", houseId)
	if err != nil {
		return nil, fmt.Errorf("%s: query flats: %w", op, err)
	}
	defer rows.Close()

	var flats []api.Flat
	for rows.Next() {
		var flat api.Flat
		if err := rows.Scan(&flat.HouseId, &flat.Id, &flat.Price, &flat.Rooms, &flat.Status); err != nil {
			return nil, fmt.Errorf("%s: scan flat: %w", op, err)
		}
		flats = append(flats, flat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration: %w", op, err)
	}

	return flats, nil
}

func (s *Storage) GetApprovedFlatsByHouseId(houseId api.HouseId) ([]api.Flat, error) {
	const op = "storage.sqlite.GetApprovedFlatsByHouseId"

	// Запрос на получение только утвержденных квартир по уникальному идентификатору дома
	rows, err := s.db.Query("SELECT house_unique_id, flat_id, price, rooms, status FROM flats WHERE house_unique_id = ? AND status = ?", houseId, api.Approved)
	if err != nil {
		return nil, fmt.Errorf("%s: query flats: %w", op, err)
	}
	defer rows.Close()

	var flats []api.Flat
	for rows.Next() {
		var flat api.Flat
		if err := rows.Scan(&flat.HouseId, &flat.Id, &flat.Price, &flat.Rooms, &flat.Status); err != nil {
			return nil, fmt.Errorf("%s: scan flat: %w", op, err)
		}
		flats = append(flats, flat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration: %w", op, err)
	}

	return flats, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}
