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

func New() *TSched {
	tsched := &TSched{
		spawnIdxs: make(map[uint64]int),
		waiters:   make(map[uint64]chan struct{}),
	}
	tsched.c = sync.NewCond(&tsched.m)
	return tsched
}

func (tsched *TSched) Yield() {
	tsched.m.Lock()
	wait := make(chan struct{})
	tsched.waiters[getGID()] = wait
	tsched.m.Unlock()
	<-wait
}

func (tsched *TSched) Go(f func()) {
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

func (tsched *TSched) Run(choose func(n int) int) {
	for {
		tsched.waitForAll()
		n := tsched.n
		if n == 0 {
			break
		}
		tsched.wake(choose(n))
	}
}

func (tsched *TSched) waitForAll() {
	tsched.m.Lock()
	for tsched.n != len(tsched.waiters) {
		tsched.c.Wait()
	}
	tsched.m.Unlock()
}

func (tsched *TSched) wake(idx int) {
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

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
