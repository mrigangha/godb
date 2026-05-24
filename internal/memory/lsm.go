package memory

import (
	"fmt"
	"os"
	"strconv"
)

type Lsm struct {
	mt    *Memtable
	f     *os.File
	tmb   map[string]struct{}
	count int
}

func NewLSM() Lsm {
	mt := NewMemtable()

	return Lsm{
		mt:    &mt,
		tmb:   make(map[string]struct{}),
		count: 1,
	}
}
func (lsm *Lsm) ShouldFlush() bool {
	return lsm.mt.size > 1000
}

func (lsm *Lsm) Insert(key string, value []byte) {
	lsm.mt.Insert(key, value)
	delete(lsm.tmb, key)
}

func (lsm *Lsm) Delete(key string) {
	lsm.tmb[key] = struct{}{}
	lsm.mt.Delete(key)
}

func (lsm *Lsm) SearchFromMemtable(key string) ([]byte, bool) {
	return lsm.mt.Search(key)
}

func (lsm *Lsm) SearchFromSStable(key string) ([]byte, bool) {
	for i := lsm.count - 1; i >= 0; i-- {
		rmap, _, err := ReadSS("ss" + strconv.Itoa(i) + ".log")
		if err != nil {
			return nil, false
		}
		val, ok := rmap[key]
		if ok {
			return val.Value, true
		}

	}
	return nil, false

}

func (lsm *Lsm) Flush() {
	f, err := os.OpenFile(
		"ss"+strconv.Itoa(lsm.count)+".log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	lsm.count += 1
	if err != nil {
		panic(err)
	}
	defer f.Close()
	old_mt := lsm.mt
	mt := NewMemtable()
	lsm.mt = &mt
	for value, _ := range lsm.tmb {
		old_mt.Insert(value, []byte(""))
	}
	data := old_mt.InOrder()
	for _, value := range data {
		if _, ex := lsm.tmb[value.Key]; ex == false {
			WriteToSS(f, SSRecord{
				Op:    SS_SET,
				Key:   value.Key,
				Value: value.Data,
			})
		} else {
			WriteToSS(f, SSRecord{
				Op:    SS_DEL,
				Key:   value.Key,
				Value: value.Data,
			})
		}
	}
	lsm.tmb = make(map[string]struct{})
	if _, err := os.Stat("wal.log"); err == nil {
		os.Remove("wal.log")
	}

}

func (lsm *Lsm) IsMergeReqd() bool {
	return lsm.count > 10
}

func (lsm *Lsm) Merge() {
	for i := lsm.count - 1; i >= 0; i-- {
		rmap, rlist, err := ReadSS("ss" + strconv.Itoa(i) + ".log")
		if err != nil {
			return nil, false
		}

		for _, record := range rlist {

			fmt.Println(record.Key)
			fmt.Println(record.Value)
		}

	}
}
