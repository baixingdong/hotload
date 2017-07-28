package hotload

import (
	"fmt"
	"sync/atomic"
	"time"
)

// doublebuffer用来进行buffer存储和切换的管理。

// Buffer -- 存储接口
type Buffer struct {
	Interface interface{} // 接口
	ServerNum int32       // 当前调用次数
}

// DoubleBuffer -- 双buffer存储结构
type DoubleBuffer struct {
	buffers [2]*Buffer // 两个Buffer结构
	pos     int32      // 当前引用的位置
}

// NewDoubleBuffer -- 创建一个双buffer结构
func NewDoubleBuffer() (db *DoubleBuffer) {
	db = &DoubleBuffer{
		buffers: [2]*Buffer{nil, nil},
		pos:     -1,
	}
	return
}

// AddBuffer -- 指定位置添加一个buffer
func (db *DoubleBuffer) AddBuffer(i interface{}, pos int32) (err error) {
	if pos < 0 || pos > 1 {
		err = fmt.Errorf("wrong pos to add buffer")
		return
	}
	db.buffers[pos] = &Buffer{
		Interface: i,
		ServerNum: 0,
	}
	if db.buffers[1-pos] != nil {
		go func() {
			for {
				sn := atomic.LoadInt32(&db.buffers[1-pos].ServerNum)
				if sn != 0 {
					time.Sleep(time.Millisecond * 50) // 50毫秒判定间隔
					continue
				}
				db.buffers[1-pos] = nil
				return
			}
		}()
	}
	err = nil
	return
}

// Store -- 存储一个接口，不会触发等待
func (db *DoubleBuffer) Store(i interface{}) (err error) {
	var np int32 = 0  // 待插入位置
	if db.pos != -1 { // 非空状态
		np = 1 - db.pos
		if db.buffers[np] != nil && db.buffers[np].ServerNum != 0 {
			err = fmt.Errorf("buffer still in used")
			return
		}
	}

	// 安全替换
	err = db.AddBuffer(i, np)
	if err != nil {
		return
	}
	db.pos = np
	return
}

// Load -- 获取当前工作中的buffer
func (db *DoubleBuffer) Load() (t *Buffer) {
	if db.pos == -1 {
		t = nil
		return
	}
	t = db.buffers[db.pos]
	atomic.AddInt32(&(t.ServerNum), 1)
	return
}

// Close 关闭当前引用的buffer
func (b *Buffer) Close() {
	sn := atomic.LoadInt32(&(b.ServerNum))
	if sn > 0 {
		atomic.AddInt32(&(b.ServerNum), -1)
	}
	return
}
