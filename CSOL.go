package csol

import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

// todo 跑一下 race detector

// CSOL is concurrently secure ordered list
type CSOL struct {
	head   *intNode
	length int64
}

type intNode struct {
	value  int
	marked uint32
	mu     sync.Mutex
	next   *intNode
}

func newIntNode(value int) *intNode {
	return &intNode{value: value, next: nil}
}

// loadNext 原子地返回下一个节点的地址
func (n *intNode) loadNext() *intNode {
	// unsafe.Pointer的使用见：https://www.jianshu.com/p/7c8e395b2981
	//return (*intNode)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(n.next))))
	return (*intNode)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&n.next))))
}

// storeNext 原子地存储下一个节点的地址
func (n *intNode) storeNext(next *intNode) {
	//atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(n.next)), unsafe.Pointer(next))
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&n.next)), unsafe.Pointer(next))
}

func NewInt() *CSOL {
	return &CSOL{head: newIntNode(0), length: 0}
}

// Len 返回有序链表的元素个数
func (c *CSOL) Len() int {
	return int(atomic.LoadInt64(&c.length))
}

// String 以字符串形式返回链表的内容
func (c *CSOL) String() string {
	a := c.head
	s := ""
	for a.loadNext() != nil {
		s = fmt.Sprintf("%s -> %v", s, a.loadNext().value)
		a = a.loadNext()
	}
	return fmt.Sprintf("# %s #", s)
}

// Slices 以slice形式返回链表的内容
func (c *CSOL) Slices() []int {
	a := c.head
	var s []int
	for a.loadNext() != nil {
		s = append(s, a.loadNext().value)
		a = a.loadNext()
	}
	return s
}

// Insert 插入一个元素，如果此操作成功插入一个元素，则返回 true，否则返回 false
func (c *CSOL) Insert(value int) bool {
	for {
		a := c.head
		b := a.loadNext()

		// 找到a.next > value的点
		for b != nil && b.value < value {
			a = b
			b = b.loadNext()
		}

		// 难点：要注意，已经存在的节点不能再次插入
		if b != nil && b.value == value {
			if atomic.LoadUint32(&b.marked) == 1 {
				continue
			}
			return false
		}

		// 锁定节点A
		a.mu.Lock()
		if a.next != b || a.marked == 1 {
			a.mu.Unlock()
			continue
		}

		x := newIntNode(value)
		x.next = b
		a.storeNext(x)
		atomic.AddInt64(&c.length, 1)
		a.mu.Unlock()
		return true
	}
}

// Delete 删除一个元素，如果此操作成功删除一个元素，则返回 true，否则返回 false
func (c *CSOL) Delete(value int) bool {
	for {
		a := c.head
		b := a.loadNext()

		// 找到a.next.value >= value的点
		for b != nil && b.value < value {
			a = b
			b = b.loadNext()
		}

		// 如果不存在>= value的点 或 这个点大于value，则返回false
		if b == nil || b.value != value {
			return false
		}

		b.mu.Lock()
		if b.marked == 1 {
			b.mu.Unlock()
			continue
		}
		a.mu.Lock()
		if a.next != b || a.marked == 1 {
			a.mu.Unlock()
			b.mu.Unlock()
			continue
		}
		atomic.StoreUint32(&b.marked, 1)
		a.storeNext(b.next)
		atomic.AddInt64(&c.length, -1)
		a.mu.Unlock()
		b.mu.Unlock()
		return true
	}

}

// Contains 检查一个元素是否存在，如果存在则返回 true，否则返回 false
func (c *CSOL) Contains(value int) bool {
	a := c.head.loadNext()
	for a != nil && a.value < value {
		a = a.loadNext()
	}
	if a == nil || a.value > value {
		return false
	}
	return atomic.LoadUint32(&a.marked) == 0
}

// Range 遍历此有序链表的所有元素，如果 f 返回 false，则停止遍历
func (c *CSOL) Range(f func(value int) bool) {
	a := c.head.loadNext()
	for a != nil {
		if atomic.LoadUint32(&a.marked) == 1 {
			a = a.loadNext()
			continue
		}
		if !f(a.value) {
			return
		}
		a = a.loadNext()
	}
}
