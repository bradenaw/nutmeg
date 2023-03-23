package tsched

import (
	"bytes"
	"runtime"
	"sort"
	"strconv"
	"sync"
)

type TSched struct {
	m          sync.Mutex
	c          *sync.Cond
	n          int
	spawnIdxs  map[uint64]int
	bySpawnIdx []uint64
	waiters    map[uint64]chan struct{}
}

func (tsched *TSched) Yield() {
	tsched.m.Lock()
	wait := make(chan struct{})
	tsched.waiters[getGID()] = wait
	tsched.m.Unlock()
	<-wait
}

func (tsched *TSched) Spawn(f func()) {
	tsched.Yield()

	tsched.m.Lock()
	spawnIdx := tsched.n
	tsched.n++
	tsched.m.Unlock()

	go func() {
		gid := getGID()
		tsched.m.Lock()
		tsched.spawnIdxs[gid] = spawnIdx
		tsched.bySpawnIdx = nil
		tsched.m.Unlock()
		tsched.Yield()

		f()

		tsched.m.Lock()
		tsched.n--
		delete(tsched.spawnIdxs, gid)
		tsched.bySpawnIdx = nil
		tsched.m.Unlock()
	}()
}

func (tsched *TSched) WaitForAll() {
	tsched.m.Lock()
	for tsched.n != len(tsched.waiters) {
		tsched.c.Wait()
	}
	tsched.m.Unlock()
}

func (tsched *TSched) N() int {
	tsched.m.Lock()
	defer tsched.m.Unlock()
	return tsched.n
}

func (tsched *TSched) Wake(idx int) {
	tsched.m.Lock()
	if len(tsched.bySpawnIdx) == 0 {
		tsched.bySpawnIdx = make([]uint64, 0, len(tsched.spawnIdxs))
		for gid := range tsched.spawnIdxs {
			tsched.bySpawnIdx = append(tsched.bySpawnIdx, gid)
		}
		sort.Slice(tsched.bySpawnIdx, func(i, j int) bool {
			return tsched.spawnIdxs[tsched.bySpawnIdx[i]] < tsched.spawnIdxs[tsched.bySpawnIdx[j]]
		})
	}
	wakeGID := tsched.bySpawnIdx[idx]
	close(tsched.waiters[wakeGID])
	delete(tsched.waiters, wakeGID)
	tsched.m.Unlock()
}

func Run(tsched *TSched, choose func(n int) int) {
	for {
		tsched.WaitForAll()
		n := tsched.N()
		if n == 0 {
			break
		}
		tsched.Wake(choose(n))
	}
}

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
