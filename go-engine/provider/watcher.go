package provider

import (
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher theo dõi các file config và gọi callback khi có thay đổi
type Watcher struct {
	fw       *fsnotify.Watcher
	onChange func(path string)
	once     sync.Once
	done     chan struct{}
}

// NewWatcher tạo watcher mới với callback
func NewWatcher(onChange func(path string)) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		fw:       fw,
		onChange: onChange,
		done:     make(chan struct{}),
	}
	go w.loop()
	return w, nil
}

// Watch thêm file vào danh sách theo dõi
func (w *Watcher) Watch(path string) error {
	return w.fw.Add(path)
}

// Unwatch bỏ theo dõi một file
func (w *Watcher) Unwatch(path string) error {
	return w.fw.Remove(path)
}

// Close dừng watcher, an toàn khi gọi nhiều lần
func (w *Watcher) Close() {
	w.once.Do(func() {
		close(w.done)
		w.fw.Close()
	})
}

// loop là goroutine xử lý filesystem events
func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.fw.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if w.onChange != nil {
					w.onChange(event.Name)
				}
			}
		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			log.Printf("[watcher] lỗi fsnotify: %v", err)
		}
	}
}
