## 文件说明

### 简单的 publish/subscribe

- simple_send.py
- simple_receive.py

这两个文件简单演示一下最简单的 **发送** 和 **接受** 逻辑是怎么写的

### eventloop 实现方法

- complete_receive.py

这个文件演示了如何使用 eventloop 实现读取逻辑

### Hub 实现方法

- hub_receive.py

这个文件演示了如何使用 Hub 实现读取逻辑

### 个人看法

其实无论是哪种方式，核心都是 Connection，所以，如果需要解析如何实现的，不妨以这种顺序查看：

1. 查看 connection 的大体实现逻辑
2. 查看 eventloop 如何对 connection 生效
3. 查看 Hub 是不是 eventloop 的一层封装

