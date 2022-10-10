# 基本理念


# Raft

## 基本概述

raft服务器三种角色：leader、candidate、follower

持久状态：currentTerm、voteFor、log[]

易失性状态：commitIndex、lastApplied

领导人易失性状态：nextIndex[]、matchIndex[]

基本原则：

+ 如果commitIndex>lastApplied，那么lastApplied递增到commitIndex，同时应用相应log到状态机

+ 如果term>currentTerm，那么清空voteFor，将currentTerm修改为term，将角色转换为follower



### 发起选举RequestVote

参数：term candidateId lastLogIndex lastLogTerm

返回值：term voteGranted

follower选举时间超时后，变为candidate，currentTerm+1，投票给自己，更新定时器。同步给其他所有节点发送请求投票。若：

+ 超过半数节点赞成，变为leader，立即同步发送心跳包

+ 若term>currentTerm，变为followers，清空voteFor

+ 选举时间过，没有任何leader当选，继续开启新一轮选举

收到请求投票：

+ 如果term>currentTerm，那么清空voteFor，将currentTerm修改为term，将角色转换为follower

+ 若满足投票篇条件（voteFor为空，lastLogIndex>mine && lastLogTerm==mine || lastLogTerm>mine, 投给它


### 同步日志AppendEntries

参数：term leaderId prevLogIndex prevLogTerm entries[] leaderCommit

返回值：term success

成为leader后，立即发送，或者收到用户端的log，或者心跳包超时后同步发送给所有其他节点

+ 若term>currentTerm，变为followers

+ 成功则检查是否能更新commitIndex，能我们则将其应用到状态机

+ 失败则往前发送

收到同步日志：

+ 若term>=currentTerm，变为followers，清空voteFor，返回失败

+ 若满足条件（prevLogIndex==mine && prevLogTerm == mine），添加到此之后，返回成功

+ 否则返回失败

+ 检查是否能更新commitIndex，能我们则将其应用到状态机



### 安全性问题论证

ps：领导人不能直接提交之前任期的条目

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

### raft优化
