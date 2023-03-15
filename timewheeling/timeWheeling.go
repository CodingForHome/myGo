package main

import (
	"fmt"
	"strconv"
	"time"
)

type jobFunc func()

type TimerTask struct {
	interval int     // 执行间隔
	callback jobFunc // 回调函数
}

func NewTimerTask(interval int, callback jobFunc) TimerTask {
	return TimerTask{interval, callback}
}

type TimeWheel struct {
	slots   [60][]TimerTask // 长度为60的时间槽数组
	tick    int             // 时间轮转动粒度
	current int             // 当前的时间槽
}

func NewTimeWheel(tick int) *TimeWheel {
	return &TimeWheel{tick: tick, current: 0}
}

func (tw *TimeWheel) addTask(task TimerTask) {
	slotIndex := (tw.current + task.interval) % 60
	tw.slots[slotIndex] = append(tw.slots[slotIndex], task)
}

func (tw *TimeWheel) advance() {
	tasks := tw.slots[tw.current]
	fmt.Println("当前时间槽：", tw.current)
	fmt.Println("执行任务数：", len(tasks))
	for _, task := range tasks {
		task.callback()
		tw.addTask(task)
	}
	tw.slots[tw.current] = []TimerTask{}
	tw.current = (tw.current + 1) % 60
}

func main() {
	tick := 1 // 时间轮转动粒度
	tw := NewTimeWheel(tick)

	// 模拟添加定时任务
	for i := 0; i < 10; i++ {
		index := i + 1
		interval := index * tick // 执行间隔
		task := NewTimerTask(interval, func() {
			fmt.Println("任务" + strconv.Itoa(index) + "执行了")
		})
		tw.addTask(task)
	}

	// 模拟时间轮转动
	for i := 0; i < 120; i++ {
		time.Sleep(time.Second * time.Duration(tick))
		tw.advance()
	}
}
