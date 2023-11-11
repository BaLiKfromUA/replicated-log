package util

import (
	"log"
	"sync"
)

// USE THIS CODE ONLY FOR SYSTEM TESTING!

type BrokenSecondaryEmulator struct {
	mu             *sync.Mutex
	shouldWait     bool
	shouldWaitCond *sync.Cond
	waitCnt        int
	waitCntCond    *sync.Cond
}

func NewBrokenSecondaryEmulator() *BrokenSecondaryEmulator {
	locker := sync.Mutex{}
	return &BrokenSecondaryEmulator{
		mu:             &locker,
		shouldWait:     false,
		shouldWaitCond: &sync.Cond{L: &locker},
		waitCnt:        0,
		waitCntCond:    &sync.Cond{L: &locker},
	}
}

func (emulator *BrokenSecondaryEmulator) BlockRequestIfNeeded() {
	emulator.mu.Lock()
	defer emulator.mu.Unlock()

	if emulator.shouldWait {
		log.Printf("[BROKEN SECONDARY] Emulation is enabled! Waiting...")
		emulator.waitCnt++

		for emulator.shouldWait {
			emulator.shouldWaitCond.Wait()
		}

		emulator.waitCnt--
		log.Printf("[BROKEN SECONDARY] Back to normal life...")
		emulator.waitCntCond.Broadcast()
	}
}

func (emulator *BrokenSecondaryEmulator) ChangeMode(shouldWait bool) {
	emulator.mu.Lock()
	defer emulator.mu.Unlock()
	log.Printf("[BROKEN SECONDARY] Mode: %t\n", shouldWait)
	emulator.shouldWait = shouldWait

	if !emulator.shouldWait {
		emulator.shouldWaitCond.Broadcast()

		for emulator.waitCnt > 0 {
			emulator.waitCntCond.Wait()
		}
	}
}

func (emulator *BrokenSecondaryEmulator) IsShouldWait() bool {
	emulator.mu.Lock()
	defer emulator.mu.Unlock()

	return emulator.shouldWait
}
