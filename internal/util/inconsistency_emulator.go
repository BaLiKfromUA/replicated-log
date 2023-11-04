package util

import (
	"log"
	"sync"
)

// USE THIS CODE ONLY FOR SYSTEM TESTING!

type InconsistencyEmulator struct {
	mu             *sync.Mutex
	shouldWait     bool
	shouldWaitCond *sync.Cond
	waitCnt        int
	waitCntCond    *sync.Cond
}

func NewInconsistencyEmulator() *InconsistencyEmulator {
	locker := sync.Mutex{}
	return &InconsistencyEmulator{
		mu:             &locker,
		shouldWait:     false,
		shouldWaitCond: &sync.Cond{L: &locker},
		waitCnt:        0,
		waitCntCond:    &sync.Cond{L: &locker},
	}
}

func (emulator *InconsistencyEmulator) BlockRequestIfNeeded() {
	emulator.mu.Lock()
	defer emulator.mu.Unlock()

	if emulator.shouldWait {
		log.Printf("Inconsistency emulation is enabled! Waiting...")
		emulator.waitCnt++

		for emulator.shouldWait {
			emulator.shouldWaitCond.Wait()
		}

		emulator.waitCnt--
		log.Printf("Back to normal life...")
		emulator.waitCntCond.Broadcast()
	}
}

func (emulator *InconsistencyEmulator) ChangeMode(shouldWait bool) {
	emulator.mu.Lock()
	defer emulator.mu.Unlock()
	log.Printf("Inconsistency Mode: %t\n", shouldWait)
	emulator.shouldWait = shouldWait

	if !emulator.shouldWait {
		emulator.shouldWaitCond.Broadcast()

		for emulator.waitCnt > 0 {
			emulator.waitCntCond.Wait()
		}
	}
}

func (emulator *InconsistencyEmulator) IsShouldWait() bool {
	emulator.mu.Lock()
	defer emulator.mu.Unlock()

	return emulator.shouldWait
}
