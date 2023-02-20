// @@
// @ Author       : Eacher
// @ Date         : 2023-02-20 08:50:39
// @ LastEditTime : 2023-02-20 08:56:21
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : Linux inotify 使用例子
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /goinotify/inotify_test.go
// @@
package inotify_test

import (
	"syscall"
	"time"
	"fmt"
	"github.com/20yyq/goinotify/inotify"
)

func TestInotify() {
	w, err := inotify.NewWatcher()
	if err != nil {
		fmt.Println("NewWatcher err", err)
		return
	}
	w.AddWatch("/temp", syscall.IN_OPEN|syscall.IN_CLOSE|syscall.IN_DELETE|syscall.IN_DELETE_SELF|syscall.IN_CREATE|syscall.IN_IGNORED|syscall.IN_MODIFY|syscall.IN_MOVE|syscall.IN_MOVE_SELF|syscall.IN_MOVED_FROM|syscall.IN_MOVED_TO|syscall.IN_MOVE_SELF|syscall.IN_ATTRIB)
	fmt.Println("start")
	for {
		ws, err := w.WaitEvent()
		if err != nil {
			fmt.Println("err", err)
			time.Sleep(time.Millisecond*300)
			continue
		}
		fmt.Println("WaitEvent:", ws.Mask, ws.FileName, ws.GetEventName())
	}
	fmt.Println("end")
}