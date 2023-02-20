//go:build linux
// +build linux
// @@
// @ Author       : Eacher
// @ Date         : 2023-02-20 08:45:05
// @ LastEditTime : 2023-02-20 09:38:14
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : Linux inotify 文件监听功能
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /goinotify/inotify/inotify.go
// @@
package inotify

import (
	"os"
	"unsafe"
	"sync"
	"syscall"
	"fmt"
	"errors"
)

type Watcher struct {
	inotifyFD 	int
	epollFD 	int

	watchMap 	map[uint32]*WatchSingle
	eventBuffer [syscall.SizeofInotifyEvent*4096]byte
	bufferItem 	uint32

	mutex   	sync.Mutex
	cond   		*sync.Cond
	wait   		bool
	closes 		bool
}

type WatchSingle struct {
	path 		string
	isDir 		bool
	watchId 	uint32
	flags 		uint32
	watch 		*Watcher
	remove 		bool

	FileName 	string
	Mask 		uint32
}

func (ws WatchSingle) GetEventName() string {
	switch {
	case ws.Mask&syscall.IN_DELETE_SELF == syscall.IN_DELETE_SELF:
		if ws.watch != nil {
			ws.watch.watchMap[ws.watchId].remove = true
		}
		return "DELETE_SELF"
	case ws.Mask&syscall.IN_MOVE_SELF == syscall.IN_MOVE_SELF:
		if ws.watch != nil {
			ws.watch.watchMap[ws.watchId].remove = true
			if _, err := syscall.InotifyRmWatch(ws.watch.inotifyFD, ws.watchId); err != nil {
				fmt.Println("Undeserved errors occur", err)
			}
		}
		return "MOVE_SELF"
	case ws.Mask&syscall.IN_CREATE == syscall.IN_CREATE:
		return "CREATE"
	case ws.Mask&syscall.IN_DELETE == syscall.IN_DELETE:
		return "DELETE"
	case ws.Mask&syscall.IN_OPEN == syscall.IN_OPEN:
		return "OPEN"
	case ws.Mask&syscall.IN_CLOSE == syscall.IN_CLOSE:
		return "CLOSE"
	case ws.Mask&syscall.IN_CLOSE_WRITE == syscall.IN_CLOSE_WRITE:
		return "CLOSE_WRITE"
	case ws.Mask&syscall.IN_CLOSE_NOWRITE == syscall.IN_CLOSE_NOWRITE:
		return "CLOSE_NOWRITE"
	case ws.Mask&syscall.IN_MOVE == syscall.IN_MOVE:
		return "MOVE"
	case ws.Mask&syscall.IN_MOVED_FROM == syscall.IN_MOVED_FROM:
		return "MOVED_FROM"
	case ws.Mask&syscall.IN_MOVED_TO == syscall.IN_MOVED_TO:
		return "MOVED_TO"
	case ws.Mask&syscall.IN_MODIFY == syscall.IN_MODIFY:
		return "MODIFY"
	case ws.Mask&syscall.IN_ATTRIB == syscall.IN_ATTRIB:
		return "ATTRIB"
	case ws.Mask&syscall.IN_IGNORED == syscall.IN_IGNORED:
		if ws.watch != nil && ws.watch.watchMap[ws.watchId].remove {
			delete(ws.watch.watchMap, ws.watchId)
		}
		return "REMOVE"
	}
	return "ERROR"
}

func (w *Watcher) AddWatch(path string, flags uint32) error {
	wd, err := syscall.InotifyAddWatch(w.inotifyFD, path, flags|syscall.IN_MASK_ADD)
	if err == nil {
		ws, ok := w.watchMap[uint32(wd)]
		if !ok {
			info, _ := os.Stat(path)
			ws = &WatchSingle{watch: w, path: path, isDir: info.IsDir(), watchId: uint32(wd), flags: flags}
			if ws.isDir {
				ws.path += string(os.PathSeparator)
			}
			w.watchMap[uint32(wd)] = ws
		}
		ws.flags |= flags
	}
	return err
}

func (w *Watcher) WaitEvent() (WatchSingle, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.bufferItem == 0 {
		if w.closes {
			return WatchSingle{}, errors.New("The Watcher is closes")
		}
		w.wait = true
		go w.epollWait()
		w.cond.Wait()
		w.wait = false
	}
	offset := uint32(syscall.SizeofInotifyEvent)
	if offset > w.bufferItem {
		return WatchSingle{}, errors.New("The event bufferItem Cross Lines")
	}
	event := (*syscall.InotifyEvent)(unsafe.Pointer(&w.eventBuffer[0]))
	ws, ok := w.watchMap[uint32(event.Wd)]
	if !ok {
		return WatchSingle{}, errors.New("The monitored directory or file has been deleted or renamed")
	}
	ws.Mask = event.Mask
	ws.FileName = ws.path
	if 0 < event.Len {
		ws.FileName += string(w.eventBuffer[offset:offset+event.Len])
		offset += event.Len
	}
	copy(w.eventBuffer[0:], w.eventBuffer[offset:])
	w.bufferItem -= offset
	return *ws, nil
}

func (w *Watcher) epollWait() {
	eventSlice := make([]syscall.EpollEvent, 3)
	n, err := syscall.EpollWait(w.epollFD, eventSlice, -1)
	if n == -1 {
		w.mutex.Lock()
		if err != syscall.EINTR {
			w.closes = true
			syscall.Close(w.inotifyFD)
		}
		if w.wait {
			w.cond.Signal()
		}
		w.mutex.Unlock()
		return
	}
	var wait bool
	for _, e := range eventSlice[:n] {
		switch {
		case e.Events&syscall.EPOLLHUP != 0:
			fallthrough
		case e.Events&syscall.EPOLLERR != 0:
			fallthrough
		case e.Events&syscall.EPOLLIN != 0:
			if e.Fd != int32(w.inotifyFD) {
				fmt.Println("The inotify fd not event fd")
				break
			}
			w.mutex.Lock()
			wait = w.wait
			if n, err := syscall.Read(w.inotifyFD, w.eventBuffer[w.bufferItem:]); err == nil {
				w.bufferItem += uint32(n)
			}
			w.mutex.Unlock()
		default:
			fmt.Println("Events Unknown")
		}
	}
	if wait {
		w.cond.Signal()
	}
}

func (w *Watcher) Close() {
	if w.inotifyFD != -1 {
		syscall.Close(w.inotifyFD)
	}
	if w.epollFD != -1 {
		syscall.Close(w.epollFD)
	}
}

func NewWatcher() (*Watcher, error) {
	w := &Watcher{inotifyFD: -1, epollFD: -1, watchMap: make(map[uint32]*WatchSingle)}
	w.inotifyFD, _ = syscall.InotifyInit1(syscall.IN_CLOEXEC)
	if w.inotifyFD == -1 {
		return nil, errors.New("The inotify cannot create")
	}
	w.epollFD, _ = syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if w.epollFD == -1 {
		syscall.Close(w.inotifyFD)
		return nil, errors.New("The epoll cannot create")
	}
	if err := syscall.EpollCtl(w.epollFD, syscall.EPOLL_CTL_ADD, w.inotifyFD, &syscall.EpollEvent{Fd: int32(w.inotifyFD), Events: syscall.EPOLLIN}); err != nil {
		syscall.Close(w.inotifyFD)
		syscall.Close(w.epollFD)
		return nil, err
	}
	w.cond = sync.NewCond(&w.mutex)
	return w, nil
}
