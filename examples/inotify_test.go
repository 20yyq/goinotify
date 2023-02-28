// @@
// @ Author       : Eacher
// @ Date         : 2023-02-20 08:50:39
// @ LastEditTime : 2023-02-28 10:10:49
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : Linux inotify 使用例子
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /inotify/examples/inotify_test.go
// @@
package inotify_test

import (
	"testing"
	"time"
	"github.com/20yyq/inotify"
)

func TestLinux(t *testing.T) {
	w, err := inotify.NewWatcher()
	if err != nil {
		t.Log("NewWatcher err", err)
		return
	}
	// err = w.AddWatch(`/mnt/veryark/develop_serial/test.go`, inotify.IN_DELETE|inotify.IN_CREATE|inotify.IN_MODIFY|inotify.IN_ATTRIB|inotify.IN_CLOSE_WRITE)
	err = w.AddWatch(`/mnt/veryark/develop_serial`, inotify.IN_OPEN|inotify.IN_DELETE|inotify.IN_CREATE|inotify.IN_MODIFY|inotify.IN_ATTRIB|inotify.IN_CLOSE_WRITE|inotify.IN_CLOSE|inotify.IN_DELETE_SELF|inotify.IN_MOVED_FROM|inotify.IN_MOVED_TO|inotify.IN_MOVE|inotify.IN_MOVE_SELF)
	t.Log("start", err)
	for {
		ws, err := w.WaitEvent()
		if err != nil {
			t.Log("WaitEvent Error", err)
			time.Sleep(time.Millisecond*300)
			continue
		}
		t.Log("WaitEvent:", ws.Mask, ws.FileName, ws.GetEventName())
	}
	t.Log("end")
}

func TestWindows(t *testing.T) {
	w, err := inotify.NewWatcher()
	if err != nil {
		t.Log("NewWatcher err", err)
		return
	}
	err = w.AddWatch(`E:\tmp`, inotify.IN_DELETE|inotify.IN_CREATE|inotify.IN_MODIFY|inotify.IN_ATTRIB|inotify.IN_CLOSE_WRITE)
	// err = w.AddWatch(`E:\tmp\adapter.js`, 0)
	t.Log("start", err)
	for {
		e, err := w.WaitEvent()
		if err != nil {
			t.Log("WaitEvent Error", err)
			time.Sleep(time.Millisecond*300)
			continue
		}
		// err = w.AddWatch(`E:\tmp`, inotify.IN_DELETE|inotify.IN_ATTRIB)
		t.Log("WaitEvent:", e.Mask, e.FileName, e.GetEventName())
	}
	t.Log("end")
}