package csol

import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

// CSOL 是并发安全的有序链表
type CSOL struct {
	head   *intNode // 头节点
	length int64    // 链表长度
}

// intNode 是链表节点
type intNode struct {
	value  int        // 节点值
	marked uint32     // 节点是否被删除的标记 0 正常 1 被删除
	mu     sync.Mutex // 节点锁
	next   *intNode   // 下一节点地址
}

// newIntNode 返回一个值为value的新节点
func newIntNode(value int) *intNode {
	return &intNode{value: value}
}

// loadNext 原子地返回下一个节点的地址
func (n *intNode) loadNext() *intNode {
	// unsafe.Pointer的使用见：https://www.jianshu.com/p/7c8e395b2981
	return (*intNode)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&n.next))))
}

// storeNext 原子地存储下一个节点的地址
func (n *intNode) storeNext(next *intNode) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&n.next)), unsafe.Pointer(next))
}

// NewInt 返回一个带头节点的空链表
func NewInt() *CSOL {
	return &CSOL{head: newIntNode(0)}
}

// Len 返回有序链表的元素个数
func (c *CSOL) Len() int {
	return int(atomic.LoadInt64(&c.length))
}

// String 以字符串形式返回链表的内容，目的是测试时查看效果
func (c *CSOL) String() string {
	a := c.head
	s := ""
	for a.loadNext() != nil {
		s = fmt.Sprintf("%s -> %v", s, a.loadNext().value)
		a = a.loadNext()
	}
	return fmt.Sprintf("# %s #", s)
}

// Slices 以slice形式返回链表的内容，目的是测试时查看效果
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
		// 第一步：找到A和B
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

		// 第二部：锁定节点A，检查AB合法性
		a.mu.Lock()
		if a.next != b || a.marked == 1 {
			a.mu.Unlock()
			continue
		}

		// 第三步：创建新节点并
		x := newIntNode(value)

		// 第四步：插入新节点
		x.next = b
		a.storeNext(x)
		atomic.AddInt64(&c.length, 1)

		// 第五步：解锁节点A
		a.mu.Unlock()
		return true
	}
}

// Delete 删除一个元素，如果此操作成功删除一个元素，则返回 true，否则返回 false
func (c *CSOL) Delete(value int) bool {
	for {
		// 第一步：找到节点A和B
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

		// 第二步：锁定节点B，检查B的合法性
		b.mu.Lock()
		if b.marked == 1 {
			b.mu.Unlock()
			continue
		}

		// 第三步：锁定节点A，检查AB合法性
		a.mu.Lock()
		if a.next != b || a.marked == 1 {
			a.mu.Unlock()
			b.mu.Unlock()
			continue
		}

		// 第四步：删除节点
		atomic.StoreUint32(&b.marked, 1)
		a.storeNext(b.next)
		atomic.AddInt64(&c.length, -1)

		// 第五步：解锁节点A和B
		a.mu.Unlock()
		b.mu.Unlock()
		return true
	}

}

// Contains 检查一个元素是否存在，如果存在则返回 true，否则返回 false
func (c *CSOL) Contains(value int) bool {
	// 第一步：找到节点X
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
	// 第一步：遍历有序链表
	a := c.head.loadNext()
	for a != nil {
		if atomic.LoadUint32(&a.marked) == 1 {
			a = a.loadNext()
			continue
		}
		// 第二步：调用 f 函数
		if !f(a.value) {
			return
		}
		a = a.loadNext()
	}
}
