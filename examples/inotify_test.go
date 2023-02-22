// @@
// @ Author       : Eacher
// @ Date         : 2023-02-20 08:50:39
// @ LastEditTime : 2023-02-22 08:09:06
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : Linux inotify 使用例子
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /inotify/examples/inotify_test.go
// @@
package inotify_test

import (
	"testing"
	"syscall"
	"time"
	"github.com/20yyq/inotify"
)

func TestInotify(t *testing.T) {
	w, err := inotify.NewWatcher()
	if err != nil {
		t.Log("NewWatcher err", err)
		return
	}
	w.AddWatch("/temp", syscall.IN_OPEN|syscall.IN_CLOSE|syscall.IN_DELETE|syscall.IN_DELETE_SELF|syscall.IN_CREATE|syscall.IN_IGNORED|syscall.IN_MODIFY|syscall.IN_MOVE|syscall.IN_MOVE_SELF|syscall.IN_MOVED_FROM|syscall.IN_MOVED_TO|syscall.IN_MOVE_SELF|syscall.IN_ATTRIB)
	t.Log("start")
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