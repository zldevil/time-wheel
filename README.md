# time-wheel

![Image text](https://raw.githubusercontent.com/zldevil/time-wheel/main/timewheel.jpg)

通过时间轮管理需要延迟触发的信号，当时间到达后通知上层调用者
支持添加单次触发的延迟通知和周期性的定时通知，为每个添加的触发生成唯一ID，可以通过该ID删除定时触发

#todo
后续可以改进成每个时间轮单独一个goroutine管理，各个时间轮通过channel进行通信 
