package wal

import (
	"encoding/binary"
	"io"
	"os"
)

const (
	WAL_SET byte = iota
	WAL_GET
	WAL_DEL
)

type WALRecord struct {
	Op    byte
	Key   string
	Value []byte
}

func WriteRecord(f *os.File, rec WALRecord) error {
	keyBytes := []byte(rec.Key)
	valueBytes := []byte(rec.Value)

	// Write operation type
	err := binary.Write(f, binary.LittleEndian, rec.Op)
	if err != nil {
		return err
	}

	// Write key length
	keyLen := uint32(len(keyBytes))
	err = binary.Write(f, binary.LittleEndian, keyLen)
	if err != nil {
		return err
	}

	// Write value length
	valueLen := uint32(len(valueBytes))
	err = binary.Write(f, binary.LittleEndian, valueLen)
	if err != nil {
		return err
	}

	// Write key bytes
	_, err = f.Write(keyBytes)
	if err != nil {
		return err
	}

	// Write value bytes
	_, err = f.Write(valueBytes)
	if err != nil {
		return err
	}

	return f.Sync()
}

func ReadRecords(path string) ([]WALRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var records []WALRecord

	for {
		var op byte

		err := binary.Read(file, binary.LittleEndian, &op)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		var keyLen uint32
		var valueLen uint32

		if err := binary.Read(file, binary.LittleEndian, &keyLen); err != nil {
			return nil, err
		}

		if err := binary.Read(file, binary.LittleEndian, &valueLen); err != nil {
			return nil, err
		}

		key := make([]byte, keyLen)
		value := make([]byte, valueLen)

		if _, err := io.ReadFull(file, key); err != nil {
			return nil, err
		}

		if _, err := io.ReadFull(file, value); err != nil {
			return nil, err
		}

		records = append(records, WALRecord{
			Op:    op,
			Key:   string(key),
			Value: value,
		})
	}

	return records, nil
}
