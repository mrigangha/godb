package internal

import (
	"encoding/json"
	"os"

	"github.com/mrigangha/nosqldb/internal/wal"
)

type Database struct {
	memory map[string][]byte
	f      *os.File
}

func (db *Database) parseAndLoadDb() {

}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func NewDatabase() Database {
	db := Database{
		f:      nil,
		memory: make(map[string][]byte),
	}
	if fileExists("wal.log") {
		var records, error = wal.ReadRecords("wal.log")
		if error == nil {
			for _, record := range records {
				if record.Op == wal.WAL_SET {
					db.memory[string(record.Key)] = record.Value
				} else if record.Op == wal.WAL_DEL {
					delete(db.memory, string(record.Key))
				}
			}
		}
	}
	f, err := os.OpenFile("wal.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		panic(err)
	}
	db.f = f
	return db
}

func (db *Database) Set(key string, value []byte) error {
	data, err := json.Marshal(value)
	if err != nil {
		panic("Invalid Type inserted")
	}
	wal.WriteRecord(db.f, wal.WALRecord{
		Op:    wal.WAL_SET,
		Key:   key,
		Value: data,
	})
	db.memory[key] = value
	defer db.f.Sync()
	return nil
}

func (db *Database) Get(key string) []byte {
	val, ok := db.memory[key]
	if ok {
		return val
	}
	return nil
}

func (db *Database) Del(key string) {
	wal.WriteRecord(db.f, wal.WALRecord{
		Op:    wal.WAL_DEL,
		Key:   key,
		Value: []byte(key),
	})
	delete(db.memory, key)
}

func (db *Database) Close() {
	db.f.Close()
}
