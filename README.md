# inotify
Linux inotify
## 简介
	这是一个可以监听多个文件和目录的项目，可以实现多路监听者。所有监听者保留监听文件或者目录的绝对路径，所有如果监听的文件或者目录本身发生了
	DELETE_SELF、MOVE_SELF这两个事件的时候，将会被完全关闭监听者。
	
# 例子
```go

func main() {
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
			fmt.Println("WaitEvent Error", err)
			time.Sleep(time.Millisecond*300)
			continue
		}
		fmt.Println("WaitEvent:", ws.Mask, ws.FileName, ws.GetEventName())
	}
	fmt.Println("end")
}

```