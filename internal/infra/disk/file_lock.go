package disk

import "sync"

var fileLocks sync.Map

func lockFile(path string) func() {
	lock, _ := fileLocks.LoadOrStore(path, &sync.Mutex{})
	mutex, _ := lock.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
}
