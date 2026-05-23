package internal

import (
	"os"

	"github.com/mrigangha/nosqldb/internal/memory"
	"github.com/mrigangha/nosqldb/internal/wal"
)

type Database struct {
	memory   map[string][]byte
	n_memory memory.Lsm
	f        *os.File
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func NewDatabase() Database {
	db := Database{
		f:        nil,
		n_memory: memory.NewLSM(),
		memory:   make(map[string][]byte),
	}
	if fileExists("wal.log") {
		var records, error = wal.ReadRecords("wal.log")
		if error == nil {
			for _, record := range records {
				if record.Op == wal.WAL_SET {

					db.n_memory.Insert(record.Key, record.Value)
				} else if record.Op == wal.WAL_DEL {
					db.n_memory.Delete(record.Key)
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
	wal.WriteRecord(db.f, wal.WALRecord{
		Op:    wal.WAL_SET,
		Key:   key,
		Value: value,
	})
	db.n_memory.Insert(key, value)
	defer db.f.Sync()
	return nil
}

func (db *Database) Get(key string) []byte {
	val, ok := db.n_memory.SearchFromMemtable(key)
	if ok {
		return val
	}
	val, ok = db.n_memory.SearchFromSStable(key)
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
	db.n_memory.Delete(key)
}

func (db *Database) Flush() {
	db.n_memory.Flush()
}

func (db *Database) Close() {
	db.f.Close()
}
