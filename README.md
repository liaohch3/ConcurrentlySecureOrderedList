# Golang 并发数据结构和算法实践 大作业

## 飞书学习笔记及踩坑见 https://bytedance.feishu.cn/docs/doccnAI0vIJp423RrXuRfESxgsh#HeOikt

## 背景
本仓库是 @zhangyunhao116 同学关于Golang 并发数据结构和算法实践课程中大作业的实现，目的是使用Golang实现一个并发安全的有序链表，其中数据严格有序并且没有重复元素。

需要完成的工作有：
- 完成插入、查询功能
- 完成删除、遍历功能
- 通过测试，并且不发生data race

## 接口
参照[单线程的有序链表实现](https://gist.github.com/zhangyunhao116/833c3113db343a660a2adb1e4c21951d)，本系统需要提供的接口有

```go
// 1. 创建有序链表
// 返回一个全新的有序链表
NewInt() *IntList

// 2. 有序链表接口
// 检查一个元素是否存在，如果存在则返回 true，否则返回 false
Contains(value int) bool

// 插入一个元素，如果此操作成功插入一个元素，则返回 true，否则返回 false
Insert(value int) bool

// 删除一个元素，如果此操作成功删除一个元素，则返回 true，否则返回 false
Delete(value int) bool

// 遍历此有序链表的所有元素，如果 f 返回 false，则停止遍历
Range(f func(value int) bool)

// 返回有序链表的元素个数
Len() int
```

## 测试
通过[链接](https://gist.github.com/zhangyunhao116/dd9f6f2f984997db18943e0e8738d257)测试
- 通过 go test (< 1s)
- 通过 go test -race (~ 70s)

## 流程图

### 插入
![](./pic/insert.png)
```mermaid
graph LR
A(开始) -->|找到A和B| B{是否找到}
B --> |否| C(结束)
B --> |是| D[锁定节点A]
D --> E{A.next!=B OR <br> A.marked}
E --> |是| A
E --> |否| F[创建新节点X<br>X.next=B<br>A.next=X]
F --> G[解锁节点A]
G --> C
```

### 删除
![](./pic/delete.png)
```mermaid
graph LR
A(开始) -->|找到A和B| B{是否找到}
B --> |否| C(结束)
B --> |是| D[锁定节点B]
D --> E{B.marked}
E --> |否| G[锁定节点A]
E --> |是| F[解锁节点B]
F --> A
G --> H{A.next!=B OR<br>A.marked}
H --> |否| J[B.marked=true<br>A.next=B.next]
H --> |是| I[解锁节点A]
I --> F
J --> K[解锁节点A <br> 解锁节点B]
K --> C
```

### 包含
![](./pic/contain.png)
```mermaid
graph LR
A(开始) -->|找到节点X| B{是否找到}
B --> |否| C[返回false]
C --> D(结束)
B --> |是| E[返回!X.marked]
E --> D
```
