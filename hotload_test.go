package hotload

import (
	"fmt"
	"testing"
	"time"
)

// test for period
type TestPeriod struct {
}

func (tp *TestPeriod) Init(interface{}) error {
	fmt.Println("I'm in initializing period.")
	return nil
}

func (tp *TestPeriod) Process(interface{}) (interface{}, error) {
	return nil, nil
}

func (tp *TestPeriod) MD5() uint64 {
	return uint64(time.Now().Unix())
}

func (tp *TestPeriod) ReloadArgument() (TargetType, interface{}) {
	return TargetTypePeriodic, time.Second * 5
}

func NewTestPeriod() HotLoader {
	return &TestPeriod{}
}

// test for watch
type TestWatch struct {
}

func (tw *TestWatch) Init(interface{}) error {
	fmt.Println("I'm in initializing watch.")
	return nil
}

func (tw *TestWatch) Process(interface{}) (interface{}, error) {
	return nil, nil
}

func (tw *TestWatch) MD5() uint64 {
	return uint64(time.Now().Unix())
}

func (tw *TestWatch) ReloadArgument() (TargetType, interface{}) {
	return TargetTypeWatch, []string{"/tmp/testwatch.tmp"}
}

func NewTestWatch() HotLoader {
	return &TestWatch{}
}

func TestTarget_Load(t *testing.T) {
	fmt.Println("start test target load...")
	targetPeriod := NewTarget(NewTestPeriod, nil)
	targetPeriod.Load()

	targetWatch := NewTarget(NewTestWatch, nil)
	targetWatch.Load()

	for {
		fmt.Printf("%+v\n%+v\n", targetPeriod.db, targetWatch.db)
		time.Sleep(time.Second * 10)
	}

}
