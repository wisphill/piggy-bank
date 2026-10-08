package state

import (
	"sync"
	"time"
)

const maxActivityEntries = 200

type ActivityEntry struct {
	At      time.Time
	Message string
}

// ActivityLog keeps the history shown on the History page and the message
// currently displayed in the toast. It outlives individual windows.
type ActivityLog struct {
	mu         sync.Mutex
	entries    []ActivityEntry
	toast      string
	toastSeq   int
	invalidate func()
}

func (l *ActivityLog) SetInvalidator(fn func()) {
	l.mu.Lock()
	l.invalidate = fn
	l.mu.Unlock()
}

// Log records msg in the history and shows it in the toast.
func (l *ActivityLog) Log(msg string) {
	l.mu.Lock()
	l.entries = append(l.entries, ActivityEntry{At: time.Now(), Message: msg})
	if len(l.entries) > maxActivityEntries {
		l.entries = l.entries[len(l.entries)-maxActivityEntries:]
	}
	l.setToastLocked(msg)
	l.mu.Unlock()
	l.notify()
}

// Status shows msg in the toast without adding it to the history.
func (l *ActivityLog) Status(msg string) {
	l.mu.Lock()
	l.setToastLocked(msg)
	l.mu.Unlock()
	l.notify()
}

// HideToastAfter hides the toast after d, unless a newer message arrived.
func (l *ActivityLog) HideToastAfter(d time.Duration) {
	l.mu.Lock()
	seq := l.toastSeq
	l.mu.Unlock()

	time.AfterFunc(d, func() {
		l.mu.Lock()
		if l.toastSeq != seq {
			l.mu.Unlock()
			return
		}
		l.toast = ""
		l.mu.Unlock()
		l.notify()
	})
}

func (l *ActivityLog) Toast() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.toast
}

// Entries returns a copy of the history, newest first.
func (l *ActivityLog) Entries() []ActivityEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]ActivityEntry, len(l.entries))
	for i, e := range l.entries {
		out[len(l.entries)-1-i] = e
	}
	return out
}

func (l *ActivityLog) Clear() {
	l.mu.Lock()
	l.entries = nil
	l.mu.Unlock()
	l.notify()
}

func (l *ActivityLog) setToastLocked(msg string) {
	l.toast = msg
	l.toastSeq++
}

func (l *ActivityLog) notify() {
	l.mu.Lock()
	fn := l.invalidate
	l.mu.Unlock()
	if fn != nil {
		fn()
	}
}
