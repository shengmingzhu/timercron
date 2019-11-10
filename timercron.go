package timercron

import (
	"github.com/shengmingzhu/timercron/taskheap"
	"github.com/shengmingzhu/timercron/timerlog"
	"github.com/shengmingzhu/timercron/timertask"
	"reflect"
	"sort"
	"sync"
	"time"
)

var t *Timer

type Timer struct {
	status uint8
	// For default, timer does not support concurrency.
	// If timerServer is setup, or outer layer will call timer's methods, you must set locker though timer.WithLocker()
	locker   *sync.Mutex
	chanStop chan struct{}

	// methods prepared by AddHandles(). Key is struct_name.method_name
	methods map[string]func()

	tasksExec map[string]struct{}
	cron      ifCron                     // this version, we use a minimum heap to implement Cron
	tasksMap  map[string]*timertask.Task // key = Task.Name, when we want to find an already pushed task, such as when we start/stop task, we need this map
}

const (
	statusStopped uint8 = iota
	statusWaitForStop
	statusRunning
)

// A ifCron represents an object that can be pop and push task.
type ifCron interface {
	Nearest() time.Duration
	Pop() *timertask.Task // pop up a task which is enabled and task.nextExecTime is smallest
	Push(task *timertask.Task)
}

func init() {
	t = New()
}

func New() *Timer {
	t := new(Timer)
	t.status = statusStopped
	t.chanStop = make(chan struct{})
	t.methods = make(map[string]func(), 0)
	t.tasksExec = make(map[string]struct{}, 0)
	t.cron = taskheap.New()
	t.tasksMap = make(map[string]*timertask.Task, 0)

	return t
}

func AddTasks(tasks ...*timertask.Task) {
	t.AddTasks(tasks...)
}

func AddHandles(in ...interface{}) {
	t.AddHandles(in...)
}

func Stop() {
	t.Stop()
}

func Start() {
	t.Start()
}

// WithLocker set locker for timer.
// If other goroutines such as timerServer call timer's methods, you must use WithLocker()
func WithLocker() {
	t.locker = &sync.Mutex{}
}
func (t *Timer) WithLocker() {
	t.locker = &sync.Mutex{}
}

// AddHandles gets all exported methods from in.
// 1. in must be exported struct.
// 2. only deals with the exported methods from exported struct.
func (t *Timer) AddHandles(in ...interface{}) {
	for _, v := range in {
		handleList := getMethods(v)
		for _, v := range handleList {
			t.methods[v.name] = v.f
		}
	}
}

type handle struct {
	name string
	f    func()
}

// getMethods uses reflect to get all exported methods from struct in.
// in must be exported struct, or getMethods will return zero.
func getMethods(in interface{}) []*handle {
	v := reflect.ValueOf(in)
	t := v.Type()

	nameHead := t.Name() + "."
	res := make([]*handle, 0)
	for i := 0; i < v.NumMethod(); i++ {
		ok := false
		h := &handle{}
		h.name = nameHead + t.Method(i).Name
		h.f, ok = v.Method(i).Interface().(func())
		if ok {
			res = append(res, h)
		}
	}

	return res
}

func (t *Timer) AddTasks(tasks ...*timertask.Task) {
	// if t is running, we stop it
	if t.status != statusStopped {
		t.Stop()
	}

	// Here t must be stopped, so if t is not empty, we can New() one
	if t.tasksMap == nil || len(t.tasksMap) > 0 {
		*t = *New()
	}

	// here we InitTick() and add tasks to tasksMap
	for _, v := range tasks {
		t.initTaskHandle(v)
		v.Init()
		t.tasksMap[v.Name] = v
	}

	// Sorted by task.tick
	// 1. After sorting, we get a minimum heap.
	//    Even if some tasks that are not enabled(task.ExpectTimes <= 0 || time.Now() > task.endTime) will not be added to task.tasksHeap,
	//    the final task.tasksHeap is still a prepared minimum heap.
	// 2. This order of joining ensures that in case if two tasks are closed, the one with a long interval must be executed later.
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].GetTick() < tasks[j].GetTick()
	})

	for _, v := range tasks {
		if !v.CheckParameters() {
			continue // This task does not expected to be execute.
		}
		v.InitNextExecTime() // Calculate after sorting to make sure the one with a long interval must be executed later
		// The previous sorting makes it unnecessary to deal with the minimum heap at this moment.
		t.cron.Push(v)
	}
}

// initTaskHandle init task.handle from timer.methods by task.HandleString
func (t *Timer) initTaskHandle(task *timertask.Task) {
	if len(task.HandleString) <= 0 {
		return
	}

	method, ok := t.methods[task.HandleString]
	if !ok {
		timerlog.Warnf("Task[%s] handleString[%s] is not exist", task.Name, task.HandleString)
		return
	}

	task.SetHandle(method)
}

func (t *Timer) lock() {
	if t.locker != nil {
		t.locker.Lock()
	}
}

func (t *Timer) unlock() {
	if t.locker != nil {
		t.locker.Unlock()
	}
}

func (t *Timer) Stop() {
	t.chanStop <- struct{}{}
	for t.status != statusStopped {
		// Do nothing, just wait for the timer to stop.
	}
}

func (t *Timer) Start() {
	if t.status == statusRunning {
		return
	}

	for t.status != statusStopped {
		// do nothing, wait to stop
	}

	t.lock()
	defer t.unlock()
	t.status = statusRunning
	go t.working()
}

func (t *Timer) execute() {
	t.lock()
	defer t.unlock()

	if t.status != statusRunning {
		return
	}

	task := t.cron.Pop()

	if task.CheckParameters() {
		task.InitNextExecTime()
		t.cron.Push(task)
		// If a concurrency conflict occurs and task.ConcurrencySupport == false, this tick will be skipped.
		if task.ConcurrencySupport || (!task.ConcurrencySupport && task.GetExecutingCount() == 0) {
			t.beforeExec(task)
			go func() {
				task.Execute()

				t.lock()
				defer t.unlock()
				t.afterExec(task)
			}()
		}
	}
}

func (t *Timer) beforeExec(task *timertask.Task) {
	_ = task.ExecutingPlusOne()
	t.tasksExec[task.Name] = struct{}{}
}

func (t *Timer) afterExec(task *timertask.Task) {
	executing := task.ExecutingMinusOne()
	if executing == 0 {
		delete(t.tasksExec, task.Name)
	}
}

func (t *Timer) working() {
	for t.status == statusRunning {
		// get a nearest task
		t.lock()
		nearest := t.cron.Nearest()
		for nearest < 0 {
			c := t.cron.Pop()
			if c == nil {
				break
			}
			nearest = t.cron.Nearest()
		}
		t.unlock()

		if nearest <= 0 {
			t.waitForStop()
			timerlog.Error("TimerCron exits from working, err: no task to scheduling, or all tasks are invalid.")
			break
		}

		var wait <-chan time.Time
		tick := nearest - time.Duration(time.Now().UnixNano())
		if tick <= 0 {
			goto execute
		}

		wait = time.Tick(tick)
		select {
		case <-t.chanStop:
			t.waitForStop()
		case <-wait:
			goto execute
		}

	execute:
		t.execute()
	}
}

func (t *Timer) waitForStop() {
	t.lock()
	t.status = statusWaitForStop
	t.unlock()

	for len(t.tasksExec) > 0 {
		// do nothing, just wait all execute goroutines stop
		time.Sleep(time.Millisecond * 100)
	}
}
