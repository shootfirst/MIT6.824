# 分布式


## 简介

#### 使用分布式的驱动力

+ 获取更高计算机性能

+ 容错

#### 分布式系统可能出现的问题

+ 网络数据丢失，重新排序，重复发送

+ 节点可能会崩溃

+ 消息被篡改（拜占庭将军问题）

#### 一致性与共识

一致性代表多个副本的数据状态，共识则是达到这个状态的手段

##### 一致性分类

+ 线性一致性：任何一次读都能读到某个数据的最近一次写的数据；系统中的所有进程，看到的操作顺序，都与全局时钟下的顺序一致；（说白了就是和单机一模一样）

+ 顺序一致性：允许对操作进行排序，就好像它们是以某种串行顺序执行一样，并要求属于某一进程的操作在排序后保持先后顺序不变

+ 因果一致性：如果两个事件有因果关系，那么在所有节点上必须观测到这个因果关系

+ 最终一致性：不保证在任意时刻任意节点上的同一份数据都是相同的，但是不同节点上的同一份数据总是在向趋同的方向变化。

#### CAP理论

+ 一致性

+ 可用性

+ 分区容错性（必须存在）

#### BASE理论

+ 基本可用

+ 软状态

+ 最终一致性





## paxos

### 角色分类

proposer acceptor learner

### paxos算法推导历程





### 三大基本原则



一个提案被选定需要被半数以上的Acceptor接受

P1：一个Acceptor必须接受它收到的第一个提案。

P2：如果某个value为v的提案被选定了，那么每个编号更高的被选定提案的value必须也是v。

P2a：如果某个value为v的提案被选定了，那么每个编号更高的被Acceptor接受的提案的value必须也是v。

P2b：如果某个value为v的提案被选定了，那么之后任何Proposer提出的编号更高的提案的value必须也是v。

P2c：对于任意的N和V，如果提案[N, V]被提出，那么存在一个半数以上的Acceptor组成的集合S，满足以下两个条件中的任意一个：

S中每个Acceptor都没有接受过编号小于N的提案。S中Acceptor接受过的最大编号的提案的value为V。

P1a：一个Acceptor只要尚未响应过任何编号大于N的Prepare请求，那么他就可以接受这个编号为N的提案。










## Raft

## 基本概述

raft服务器三种角色：leader、candidate、follower

持久状态：currentTerm、voteFor、log[]

易失性状态：commitIndex、lastApplied

领导人易失性状态：nextIndex[]、matchIndex[]

基本原则：

+ 如果commitIndex>lastApplied，那么lastApplied递增到commitIndex，同时应用相应log到状态机

+ 如果term>currentTerm，那么清空voteFor，将currentTerm修改为term，将角色转换为follower


代码总结：

在每次rpc返回后，需要检测自己term或者角色是否改变，改变，则直接置之不理！





### 发起选举RequestVote

参数：term candidateId lastLogIndex lastLogTerm

返回值：term voteGranted

##### 发送请求投票

follower选举时间超时后，变为candidate，currentTerm+1，投票给自己，更新定时器。同步给其他所有节点发送请求投票。若：

+ 超过半数节点赞成，变为leader，立即同步发送心跳包

+ 若term>currentTerm，变为followers，清空voteFor

+ 选举时间过，没有任何leader当选，继续开启新一轮选举

+ 收到其他leader的心跳，并且term大于等于自己，变为follow（在appendrpc中实现）

##### 收到请求投票：

+ 如果term>currentTerm，那么清空voteFor，将currentTerm修改为term，将角色转换为follower

+ 日志至少和自己一样新, 投给它

##### 后发起选举

注意随机化时间，防止同时发起选举





### 同步日志AppendEntries

参数：term leaderId prevLogIndex prevLogTerm entries[] leaderCommit

返回值：term success

##### 发送同步日志

成为leader后，立即发送，或者收到用户端的log，或者心跳包超时后同步发送给所有其他节点

+ 若term>currentTerm，变为followers

+ 成功则检查是否能更新commitIndex，能我们则将其应用到状态机

+ 失败则往前发送

##### 接收同步日志

+ 若term>=currentTerm，变为followers，清空voteFor，返回失败

+ 若满足条件（prevLogIndex==mine && prevLogTerm == mine），添加到此之后，检查是否能更新commitIndex，能我们则将其应用到状态机，返回成功

+ 否则返回失败，同时，为了加速，这里可以附上本次冲突日志的任期上一次任期的开头任期的index





### 安全性问题论证

##### 提交之前任期的日志（论文图8）

leaders can only commit logs with term same as his own

接下来是五大安全特性：

##### 选举安全特性

领导人只有一个

##### 领导人只附加原则

领导人只附加日志，而不删除

##### 日志匹配原则

两日志索引相同，日志相同，那么从头到此日志都相同

##### 领导人完全特性

某个日志条目在某个任期被提交，它一定会出现在更大任期号的所有领导人里

##### 状态机安全特性

若某个服务器将某个log应用到状态机，那么其他服务器不会在这个位置应用不同条目






## 工业界应用扩展

### 日志压缩

在实际中，日志增长会占据大量内存，所以需要日志压缩。一般数量到达某个阈值后压缩成快照。当leader需要同步压缩的日志时直接发送快照。

### 禅让

在实际中，有时候需要更换leader，直接停机leader会浪费，可以采取禅让机制。

+ leader向target发起一次日志同步

+ 向target发送TImeOutNowRPC，target收到之后马上开启选举

### 预投票

在实际中，暂时脱离集群的节点重新回来后会干扰到集群的运行。为了防止它使leader退位。

一个candidate必须在获得了多数赞同的情形下，才会增加自己的 term。一个节点在满足下述条件时，才会赞同一个 candidate：

+ 该 candidate 的日志足够新

+ 当前节点已经和 leader 失联（electionTimeout）

### 客户端响应

为了防止客户端超时重传，在上层需要防止重复的提交

### 配置变更

##### 变更一台

##### 变更多台



## raft优化
