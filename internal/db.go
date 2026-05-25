package internal

import (
	"errors"
	"os"
	"sync"
	"sync/atomic"

	"github.com/mrigangha/nosqldb/internal/memory"
	"github.com/mrigangha/nosqldb/internal/wal"
)

var ErrDatabaseClosed = errors.New("database is closed")

type Database struct {
	n_memory memory.Lsm

	// atomic WAL file pointer
	walFile atomic.Pointer[os.File]

	// async WAL queue
	walQueue chan wal.WALJob

	// db closed state
	closed atomic.Bool

	// memtable lock
	mu sync.RWMutex

	// WAL file lock
	walMu sync.Mutex

	// SSTable lock
	sstMu sync.RWMutex

	// worker lifecycle
	wg sync.WaitGroup
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func NewDatabase() *Database {

	db := &Database{
		n_memory: memory.NewLSM(),
		walQueue: make(chan wal.WALJob, 10000),
	}

	// WAL recovery
	if fileExists("wal.log") {

		records, err := wal.ReadRecords("wal.log")

		if err == nil {

			for _, r := range records {

				switch r.Op {

				case wal.WAL_SET:
					db.n_memory.Insert(r.Key, r.Value)

				case wal.WAL_DEL:
					db.n_memory.Delete(r.Key)
				}
			}
		}
	}

	// open WAL file
	f, err := os.OpenFile(
		"wal.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)

	if err != nil {
		panic(err)
	}

	db.walFile.Store(f)

	// start WAL worker
	db.wg.Add(1)
	go db.walWorker()

	return db
}

func (db *Database) Set(key string, value []byte) error {

	if db.closed.Load() {
		return ErrDatabaseClosed
	}

	done := make(chan error, 1)

	job := wal.WALJob{
		Record: wal.WALRecord{
			Op:    wal.WAL_SET,
			Key:   key,
			Value: value,
		},
		Done: done,
	}

	// enqueue WAL job
	db.walQueue <- job

	// wait for WAL durability
	if err := <-done; err != nil {
		return err
	}

	// insert into memtable
	db.mu.Lock()
	db.n_memory.Insert(key, value)
	db.mu.Unlock()

	return nil
}

func (db *Database) Del(key string) error {

	if db.closed.Load() {
		return ErrDatabaseClosed
	}

	done := make(chan error, 1)

	job := wal.WALJob{
		Record: wal.WALRecord{
			Op:  wal.WAL_DEL,
			Key: key,
		},
		Done: done,
	}

	// enqueue WAL job
	db.walQueue <- job

	// wait for WAL durability
	if err := <-done; err != nil {
		return err
	}

	// tombstone in memtable
	db.mu.Lock()
	db.n_memory.Delete(key)
	db.mu.Unlock()

	return nil
}

func (db *Database) Get(key string) []byte {

	// memtable lookup
	db.mu.RLock()
	val, ok := db.n_memory.SearchFromMemtable(key)
	db.mu.RUnlock()

	if ok {
		return val
	}

	// SSTable lookup
	db.sstMu.RLock()
	val, ok = db.n_memory.SearchFromSStable(key)
	db.sstMu.RUnlock()

	if ok {

		// optional cache promotion
		db.mu.Lock()
		db.n_memory.Insert(key, val)
		db.mu.Unlock()

		return val
	}

	return nil
}

func (db *Database) Flush() {

	db.sstMu.Lock()
	db.n_memory.Flush()
	db.sstMu.Unlock()
}

func (db *Database) ShouldFlush() bool {
	return db.n_memory.ShouldFlush()
}

func (db *Database) Merge() {

	db.sstMu.Lock()
	db.n_memory.Merge()
	db.sstMu.Unlock()
}

func (db *Database) ShouldMerge() bool {
	return db.n_memory.IsMergeReqd()
}

func (db *Database) Close() {

	// prevent double close
	if db.closed.Swap(true) {
		return
	}

	// stop worker gracefully
	close(db.walQueue)

	// wait for worker exit
	db.wg.Wait()

	db.walMu.Lock()

	file := db.walFile.Load()

	if file != nil {
		file.Sync()
		file.Close()
	}

	db.walMu.Unlock()
}

func (db *Database) walWorker() {
	defer db.wg.Done()

	batch := make([]wal.WALJob, 0, 100)

	flushBatch := func() {

		if len(batch) == 0 {
			return
		}

		db.walMu.Lock()

		file := db.walFile.Load()

		var batchErr error

		// write WAL batch
		for _, j := range batch {

			if err := wal.WriteRecord(file, j.Record); err != nil {
				batchErr = err
				break
			}
		}

		// fsync
		if batchErr == nil {

			if err := file.Sync(); err != nil {
				batchErr = err
			}
		}

		db.walMu.Unlock()

		// notify waiters
		for _, j := range batch {
			j.Done <- batchErr
		}

		batch = batch[:0]
	}

	for job := range db.walQueue {

		batch = append(batch, job)

	Drain:
		for len(batch) < 100 {

			select {

			case j, ok := <-db.walQueue:

				if !ok {
					break Drain
				}

				batch = append(batch, j)

			default:
				break Drain
			}
		}

		flushBatch()
	}

	// flush remaining jobs before exit
	flushBatch()
}
