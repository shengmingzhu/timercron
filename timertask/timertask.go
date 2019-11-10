package timertask

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/shengmingzhu/timercron/timerlog"
	"math"
	"sync/atomic"
	"time"
)

type Task struct {
	Name string `json:"name"`
	// concurrencySupport
	// true: When a task is executing, and the next tick is up, it will be executed concurrently.
	// false: The timer will skip this tick, if there's a task be executing.
	ConcurrencySupport bool `json:"concurrencySupport"`
	Disable            bool `json:"disable"` // Set task.disable to true will stop or disable this task.

	// Control start or stop, if needed
	StartTime   string `json:"startTime"`   // After startTime, we start the task
	EndTime     string `json:"endTime"`     // After endTime, we stop the task
	ExpectTimes uint32 `json:"expectTimes"` // When the task executed more than Task.expectTimes, we stop it

	// Control the execution interval
	// In most cases, you only need to configure one of the following parameters,
	// such as tickSecond = 10 for every 10 seconds.
	// If multiple parameters is set, we will add them together to Task.tick
	TickDay        time.Duration `json:"tickDay"`
	TickSecond     time.Duration `json:"tickSecond"`
	TickNanosecond time.Duration `json:"tickNanosecond"`

	HandleString string `json:"handleString"` // Method path, must be format "struct_name.method_name"

	handle func() // Specific function to be executed

	executedCount  uint32 // Count the number executed
	executingCount int32  // Count the number executing
	startTime      time.Duration
	endTime        time.Duration
	tick           time.Duration // the real execution interval
	nextExecTime   time.Duration
}

func (t *Task) Init() {
	t.executedCount = 0

	t.initTick()
	t.initStartTime()
	t.initEndTime()
	t.InitNextExecTime()

	t.CheckParameters()
}

// checkParameters check if the parameters are valid, if not, set task.Disable to true.
// It returns true for the enabled task, and false for the disabled task, in fact returns !t.Disable.
func (t *Task) CheckParameters() bool {
	if t.Disable {
		return !t.Disable
	}

	if (t.ExpectTimes < 0 || (t.ExpectTimes > 0 && t.ExpectTimes <= t.getExecutedCount())) ||
		t.tick <= 0 ||
		t.handle == nil ||
		t.endTime <= time.Duration(time.Now().UnixNano()) {
		t.Disable = true
	}

	return !t.Disable
}

func (t *Task) initStartTime() {
	if len(t.StartTime) <= 0 {
		t.startTime = time.Duration(math.MinInt64)
		return
	}
	var err error
	t.startTime, err = initTime(t.StartTime)
	if err != nil {
		timerlog.Errorf("Task[%s] initStartTime failed: %s", t.Name, err.Error())
	}
}

func (t *Task) initEndTime() {
	if len(t.EndTime) <= 0 {
		t.endTime = time.Duration(math.MaxInt64)
		return
	}
	var err error
	t.endTime, err = initTime(t.EndTime)
	if err != nil {
		timerlog.Errorf("Task[%s] initEndTime failed: %s", t.Name, err.Error())
	}
}

func initTime(s string) (t time.Duration, err error) {
	var timeTemplate = `2006-01-02 15:04:05`
	if len(s) > 0 {
		tm, err := time.Parse(timeTemplate, s)
		if err == nil {
			t = time.Duration(tm.UnixNano())
		} else {
			err = errors.New(fmt.Sprintf("Parse time[%s] err: %s", s, err.Error()))
		}
	}

	return t, err
}

func (t *Task) initTick() {
	t.tick = time.Hour*24*t.TickDay + time.Second*t.TickSecond + t.TickNanosecond
}

func (t *Task) InitNextExecTime() {
	if t.Disable {
		t.nextExecTime = math.MaxInt64
	} else {
		t.nextExecTime = time.Duration(time.Now().UnixNano()) + t.tick
	}
}

func (t *Task) GetTick() time.Duration {
	return t.tick
}

func (t *Task) GetNextExecTime() time.Duration {
	return t.nextExecTime
}

func (t *Task) GetName() string {
	return t.Name
}

func (t *Task) GetHandleString() string {
	return t.HandleString
}

func (t *Task) Concurrency() bool {
	return t.ConcurrencySupport
}

// ExecutingPlusOne, pluses 1 into task.executingCount, and return the new value.
// It's atomically, however, you can't trust the return value,
// because there's may be an other goroutine modified the real value when you process the return value.
func (t *Task) ExecutingPlusOne() int32 {
	return atomic.AddInt32(&t.executingCount, 1)
}

// ExecutingMinusOne, minus 1 from task.executingCount, and return the new value.
// Same as ExecutingPlusOne(), it's atomically and you can't over trust the return value.
func (t *Task) ExecutingMinusOne() int32 {
	return atomic.AddInt32(&t.executingCount, -1)
}

func (t *Task) GetExecutingCount() int32 {
	return atomic.LoadInt32(&t.executingCount)
}

func (t *Task) getExecutedCount() uint32 {
	return atomic.LoadUint32(&t.executedCount)
}

func (t *Task) Execute() {
	t.handle()
	atomic.AddUint32(&t.executedCount, 1)
}

func (t *Task) SetHandle(f func()) {
	t.handle = f
}
