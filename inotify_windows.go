//go:build windows
// +build windows
// @@
// @ Author       : Eacher
// @ Date         : 2023-02-21 09:46:27
// @ LastEditTime : 2023-02-28 10:16:06
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : windows 文件通知项目
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /inotify/inotify_windows.go
// @@
package inotify

import (
	"os"
	"unsafe"
	"syscall"
	"fmt"
	"path/filepath"
)

const bufferSize = 100

const (
	in_CLOSE 				= 0x00000000
	in_CLOSE_NOWRITE		= 0x00000000
	in_DELETE_SELF 			= 0x00000000
	in_MOVE 				= 0x00000000
	in_MOVED_FROM 			= 0x00000000
	in_MOVED_TO 			= 0x00000000
	in_MOVE_SELF 			= 0x00000000
	in_OPEN					= 0x00000000
	in_CREATE				= 0x00000000
	// in_OPEN					= syscall.FILE_NOTIFY_CHANGE_LAST_ACCESS
	// in_CREATE				= syscall.FILE_NOTIFY_CHANGE_CREATION
	// in_MOVE					= syscall.FILE_NOTIFY_CHANGE_SECURITY

	in_DELETE				= syscall.FILE_NOTIFY_CHANGE_FILE_NAME|syscall.FILE_NOTIFY_CHANGE_DIR_NAME
	in_ATTRIB				= syscall.FILE_NOTIFY_CHANGE_ATTRIBUTES
	in_MODIFY				= syscall.FILE_NOTIFY_CHANGE_SIZE
	in_CLOSE_WRITE			= syscall.FILE_NOTIFY_CHANGE_LAST_WRITE
)

type WatchSingle struct {
	path 		string
	isDir 		bool
	h 			syscall.Handle
	flags 		uint32
	watch 		*Watcher
	// remove 		bool
	buf 		[]byte
}

type EventBody struct {
	wd 			uint32
	FileName 	string
	Mask 		uint32
}

func (eb EventBody) GetEventName() string {
	switch {
	case eb.Mask == syscall.FILE_ACTION_MODIFIED:
		return "MODIFIED"
	case eb.Mask == syscall.FILE_ACTION_ADDED:
		return "ADDED"
	case eb.Mask == syscall.FILE_ACTION_REMOVED:
		return "REMOVED"
	case eb.Mask == syscall.FILE_ACTION_RENAMED_NEW_NAME:
		return "RENAMED_NEW"
	case eb.Mask == syscall.FILE_ACTION_RENAMED_OLD_NAME:
		return "RENAMED_OLD"
	}
	return "ERROR"
}

type Watcher struct {
	cphandle 	syscall.Handle
	watchMap 	map[uint32]*WatchSingle
	e 			chan *EventBody

	closes 		bool
}

func NewWatcher() (*Watcher, error) {
	var err error
	w := &Watcher{watchMap: make(map[uint32]*WatchSingle), e: make(chan *EventBody, 10)}
	w.cphandle, err = syscall.CreateIoCompletionPort(syscall.InvalidHandle, 0, 0, 1)
	if err != nil {
		return nil, fmt.Errorf("Watcher new Error: %s", err.Error())
	}
	go w.epollWait()
	return w, nil
}

// 单个文件监听暂时不支持
func (w *Watcher) AddWatch(path string, flags uint32) error {
	var err error
    if path, err = filepath.Abs(path); err != nil {
    	return err
    }
    if info, _ := os.Stat(path); info != nil {
    	if !info.IsDir() {
    		return fmt.Errorf("Watcher Dir Only")
    	}
		var h syscall.Handle
		if h, err = syscall.CreateFile(syscall.StringToUTF16Ptr(path), syscall.FILE_LIST_DIRECTORY,
			syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil,
			syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OVERLAPPED, 0); err != nil {
			return err
		}
		
		ws, ok := w.watchMap[uint32(h)]
		// TODO 暂未能实现更新事件监听mask
		if ok {
			ws.flags |= flags
			return nil
		}
		ws = &WatchSingle{watch: w, path: path, isDir: info.IsDir(), h: h, flags: flags, buf: make([]byte, bufferSize)}
		if ws.isDir {
			ws.path += string(os.PathSeparator)
		}
		if _, err = syscall.CreateIoCompletionPort(ws.h, w.cphandle, uint32(ws.h), 0); err != nil {
			syscall.CloseHandle(ws.h)
			return err
		}
		if err = syscall.ReadDirectoryChanges(ws.h, &ws.buf[0], bufferSize, true, ws.flags, nil, &syscall.Overlapped{}, 0); err != nil {
			syscall.CloseHandle(ws.h)
			return err
		}
		w.watchMap[uint32(ws.h)] = ws
		return nil
    }
    return fmt.Errorf("File or Dir not")
}

func (w *Watcher) WaitEvent() (EventBody, error) {
	if w.closes {
		return EventBody{}, fmt.Errorf("The Watcher is closes")
	}
	e, ok := <-w.e
	if e == nil && !ok{
		return EventBody{}, fmt.Errorf("The Watcher is closes")
	}
	return *e, nil
}

func (w *Watcher) epollWait() {
	var qty, key uint32
	var ov *syscall.Overlapped
	for {
		err := syscall.GetQueuedCompletionStatus(w.cphandle, &qty, &key, &ov, syscall.INFINITE)
		if err != nil {
			fmt.Println("The GetQueuedCompletionStatus error ", err)
			continue
		}
		if key == 0 {
			w.closes = true
			syscall.Close(w.cphandle)
			syscall.CancelIo(w.cphandle)
			for _, v := range w.watchMap {
				syscall.CancelIo(v.h)
				syscall.Close(v.h)
			}
			close(w.e)
			return
		}
		ws, ok := w.watchMap[key]
		if !ok {
			fmt.Println("The watchMap error ", key)
			continue
		}
		event := (*syscall.FileNotifyInformation)(unsafe.Pointer(&ws.buf[0]))
		body := &EventBody{wd: key, Mask: event.Action, FileName: ws.path}
		if ws.isDir {
			body.FileName += syscall.UTF16ToString(((*[syscall.MAX_PATH]uint16)(unsafe.Pointer(&event.FileName)))[:event.FileNameLength/2])
		}

		// 留存不超过10个缓存事件
		if len(w.e) == cap(w.e) {
			<-w.e
		}
		w.e <- body

		if err = syscall.ReadDirectoryChanges(ws.h, &ws.buf[0], bufferSize, true, ws.flags, nil, &syscall.Overlapped{}, 0); err != nil {
			fmt.Println("The ReadDirectoryChanges error ", err)
			delete(w.watchMap, key)
		}
	}
}

func (w *Watcher) Close() error {
	if !w.closes {
		return syscall.PostQueuedCompletionStatus(w.cphandle, 0, 0, nil)
	}
	return nil
}
