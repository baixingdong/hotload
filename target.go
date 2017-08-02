package hotload

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"os"
	"time"
)

// target 是作为热加载的核心组件，控制热加载的集中情形

// HotLoader -- 热加载的接口
type HotLoader interface {
	Init(interface{}) error                    // 初始化过程
	Process(interface{}) (interface{}, error)  // 处理函数
	MD5() uint64                               // 获取更新MD5的函数
	ReloadArgument() (TargetType, interface{}) // 重载参数
}

// HotLoaderCreator -- 热加载接口创建器
type HotLoaderCreator func() HotLoader

// Target -- 热加载的承载对象
type Target struct {
	creator HotLoaderCreator // 一个HotLoader的创建函数
	db      *DoubleBuffer    // 双buffer结构
	ttype   TargetType       // 热加载类型
	args    interface{}      // 加载参数
	event   chan bool        // 热加载触发
	success bool             // 是否load成功
	config  interface{}      // 配置信息
	stamp   uint64           // 戳，用来决定是否更新
}

// TargetType -- 热加载的对象类型，枚举
type TargetType int

const (
	_                  TargetType = iota // 默认值
	TargetTypePeriodic                   // 周期性自动reload
	TargetTypeListen                     // 监听触发
	TargetTypeWatch                      // 监控文件触发
)

// 这里定义几个热加载类型对应的函数
// doPeriodic 周期性reload
func doPeriodic(interval time.Duration, function func(interface{}), data interface{}) {
	tick := time.NewTicker(interval)

	for {
		select {
		case <-tick.C:
			function(data)
		}
	}
}

// doListen 监听触发
func doListen(event chan bool, function func(interface{}), data interface{}) {
	for {
		select {
		case <-event:
			function(data)
		}
	}
}

// doWatch 监控文件触发
func doWatch(paths []string, function func(interface{}), data interface{}) {
	// 如果文件不存在，先创建path
	for _, path := range paths {
		_, err := os.Stat(path)
		if err != nil && !os.IsExist(err) {
			os.Create(path)
		}
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	for _, path := range paths {
		watcher.Add(path)
	}

	for {
		select {
		case ev := <-watcher.Events:
			if ev.Op == fsnotify.Remove || ev.Op == fsnotify.Rename {
				// 如果监控到文件被删除或者被移动，更换inode
				watcher.Remove(ev.Name)
				time.Sleep(time.Millisecond * 50)
				_, err := os.Stat(ev.Name)
				if err != nil && !os.IsExist(err) {
					return
				}
				function(data)
				watcher.Add(ev.Name)
				continue
			}
			// 任何情况下触发reload
			function(data)
		}
	}

	return
}

// NewTarget -- 创建一个对象
func NewTarget(creator HotLoaderCreator, config interface{}) (target *Target) {
	target = &Target{
		creator: creator,
		db:      NewDoubleBuffer(),
		event:   make(chan bool),
		success: false,
		config:  config,
	}
	return
}

// Load -- 加载
func (target *Target) Load() (err error) {
	i := target.creator()
	err = i.Init(target.config)
	if err != nil {
		return
	}

	// 初次load应该是没有
	target.db.Store(i)
	target.success = true
	target.stamp = i.MD5()
	target.ttype, target.args = i.ReloadArgument()
	err = nil

	reloadWrapper := func(t interface{}) {
		t.(*Target).reload()
	}
	switch target.ttype {
	case TargetTypeListen:
		go doListen(target.event, reloadWrapper, target)
	case TargetTypePeriodic:
		go doPeriodic(target.args.(time.Duration), reloadWrapper, target)
	case TargetTypeWatch:
		go doWatch(target.args.([]string), reloadWrapper, target)
	}
	return
}

// Process -- 处理函数的wrapper
func (target *Target) Process(src interface{}) (dst interface{}, err error) {
	i := target.db.Load()
	defer i.Close()
	dst, err = i.Interface.(HotLoader).Process(src)
	return
}

// MD5 -- 签名函数的wrapper
func (target *Target) MD5() (md5 uint64) {
	i := target.db.Load()
	defer i.Close()
	md5 = i.Interface.(HotLoader).MD5()
	return
}

// reload -- 重载，只允许内部使用
func (target *Target) reload() (err error) {
	i := target.creator()
	md5 := target.MD5()
	if md5 == 0 { // md5为0的情况下认为是永远不更新
		err = fmt.Errorf("0 for stamp")
		return
	}
	if md5 == target.stamp { // 数据没有发生变化，不更新
		err = fmt.Errorf("no data changed")
		return
	}
	// 初始化
	e := i.Init(target.config)
	if e != nil {
		err = fmt.Errorf("init error[%s]", e.Error())
		return
	}

	// 存储
	e = target.db.Store(i)
	if e != nil {
		err = fmt.Errorf("store error[%s]", e.Error())
		return
	}
	target.stamp = md5
	err = nil
	return
}
