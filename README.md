# MIT6.824
MIT2020年秋分布式系统实验

## lab1

实验一是在单机上通过多线程实现mapreduce，对于我这种刚开始学分布式和从来没有接触过go的同学一开始会比较痛苦。首先我花了两小时去看824的第一节课，然后花了大概一小时把mapreduce
论文看了个大概，最后花了两个小时入门了go（没有入门goruntime和channel）然后在阅读实验一指导书和慢慢写代码的时候入门了一下channel和goruntime语法。渐渐地理清了思路。

数据结构：

		type Request struct {
			RQ RequestType
			JobType TaskType
			Xth     int
		}
		
		type Job struct {
			JobType TaskType
			Files   []string
			Xth     int
			Nreduce      int
		}

上述两个数据结构作为rpc的两个参数，需要注意一点就是这两个结构体的所有字段都必须以大写字母开头，否则go的底层是不会序列化成功的

		type JobInfo struct {
			jobstatus JobStatus
			start time.Time
			jobptr *Job
		}
		
这个则是记录job的所有信息，是还没有分配，分配中还是已经完成，分配出去的时间以及指向对应Job的指针。

		type Master struct {
			// Your definitions here.
			
			// these two variables are only readable
			nm int
			nr int
		
			// protect 
			mutex sync.Mutex
		
			status PhaseType
		
			MapChannel chan *Job
			ReduceChannel chan *Job
		
			mapJobInfo [] JobInfo
			reduceJobInfo [] JobInfo
		
		}

而这个则是master结构体

- status记录master的状态，一共有三种状态，处于map阶段、处于reduce阶段和结束阶段

- Mapchannel则是作为天然线程安全的job队列，存储所有的mapjob，reduce同理

- mapJobInfo则是记录mapjob的信息

### 亮点

与之前写lab不同，写824的时候，我首先是确保彻底搞清楚实验思路，脑海中明确了实验该怎么去写，然后才慢慢去写对应代码，比以前有时候仓促写代码有很大进步。然后在实现自己代码的时候
我要确保我写的代码达到的效果必须和我思考的一致，有可能会出现一些其他的bug导致和自己预期的不一样，所以我给整个实验确定了以下迭代步骤：

- 首先按照mrsequential.go文件的源码正确实现调用map和reduce处理

- 确保master和worker之间能够正常通信，也就是Request和Job结构体能够通过rpc正确传输，这里我解决了字段没有大写的bug

- 开始实现handler（供rpc远程调用的函数），按照不同的阶段，不同的请求实现之。实现之后开启多个shell跑worker程序，预期结果和想象中一致

- 实现crashhandler恢复函数，周期性调用，将发射超过10s的job重新加入到channel，这里我将worker睡眠100秒，得到证实确实将超时的job加入到了指定channel的指定位置

- 最后实现中间态文件，没有注意到前缀一词，导致文件并没有被正确的重命名，最后经过仔细阅读后解决之

可以从以上看到，这种逐步迭代验证的写代码思路和我以前是完全不一样的，这种步步为营，稳扎稳打的写代码方式我感觉很棒

以下具体说明代码实现思路，以程序执行顺序来讲解

### master

#### MakeMaster

首先是该函数被调用，该函数完成必要的初始化

+ 设置maptask数量和reducetask数量

+ 将状态设置为mapphase

+ 调用GenerateMapTask生成map任务

+ 生成CrashHandler协程，周期性检查是否有worker崩溃

+ 调用server生成server协程，底层会调用Handler函数

+ 结束，注意这里结束并不意味着master结束，留下了两个协程，一个是CrashHandler，负责崩溃检查，一个是server负责为worker的远程调用Handler函数服务

#### GenerateMapTask

负责生成map任务

+ 初始化MapChannel和mapJobInfo

+ 将任务加入到MapChannel和mapJobInfo中

#### CrashHandler

周期性检查是否有崩溃，通过时间检查来判断是否有worker崩溃，这里不需要知道哪个worker崩溃，只需要将相关超时的job重新分发即可

- 睡眠1s

- 获取锁

- 若master状态为 mapphase，检查mapJobInfo，通过检查是否状态为processing并且处理时间超过10s的job，有，我们则将状态恢复为waiting，然后将其加入对应channel

- 若master状态为 reducephase，同上

- 若状态为 finishphase，表示所有任务完成，此时退出循环

- 释放锁，继续循环

#### Handler

这个是负责分配任务的函数，通过接收worker通过rpc传入的参数判断请求类型

- 获取锁

- 若是请求任务分配，通过目前的状态，判断应该分配哪种任务

  * 若是map或者是reduce

    + 判断对应channel长度是否为0，为0则分配任务为等待，让worker睡眠2s
  
    + 不为0，则发送job，记录时间戳，状态修改为processing
  
  * 若是finishphase

    + 表示所有任务已经完成，发送kill任务让worker进程结束
  
- 若是提交任务

  * 若当前是mapphase，那么我们只能接收map任务提交（其实也只有map任务提交），若当前是reducephase，则我们只能接收reduce任务提交，有可能会出现map提交，我们不能接收
  
  * 接收表示，将对应任务状态修改为finished
  
  * 接收完成之后，我们调用CheckAndGrow检查是否能修改master状态
  
#### CheckAndGrow

检查是否能修改master状态

- 若当前状态为mapphase，则我们检查mapJobInfo，若全部是finished我们调用GenerateReduceTask准备reduce任务，同时修改状态为reducephase

- 若状态为reducephase，我们检查reduceJobInfo，若全部是finished我们将状态修改为finishphase，表示结束

#### GenerateReduceTask

和上面类似

#### Done

判断能否结束，会被上层周期性调用，若为true则退出直接，我们直接判断状态是否为finishphase即可

### worker

worker启动后调用Worker函数，该函数结束则worker结束

### Worker

- 将请求字段设置为请求获取任务

- 调用call通过rpc获取master服务

- master返回任务，判断任务类型

  * map，调用DoMap函数
  
  * reduce，调用DoReduce任务
  
  * waiting，睡眠2s
  
  * kill，结束循环，退出函数
  
- 无限循环之

#### DoMap

这里代码基本上是参照mrsequential.go。

- 调用mapf将键值对收集到内存切片（实际上一个是收集到磁盘中，内存是没有这么大的）

- 排序

- 准备内存哈希表，将应该放置到同一个中间文件的kv对放在一个bucket

- 调用encoder，将键值对写入指定中间文件（先创建）

- 注意首先写入临时文件，最后原子改名为指定中间文件，这是为了防止worker在写文件时忽然崩溃

- 最后发送job完成的信息给master

#### DoReduce

和上面几乎一样，不赘述


## lab2

实验二是实现raft算法。很虐的一个实验。首先这是raft结构体，变量名和raft论文中的一样。

	type Raft struct {
		mu        sync.Mutex          // Lock to protect shared access to this peer's state
		peers     []*labrpc.ClientEnd // RPC end points of all peers
		persister *Persister          // Object to hold this peer's persisted state
		me        int                 // this peer's index into peers[]
		dead      int32               // set by Kill()
	
	
		applyCh       chan ApplyMsg
		applyCond     *sync.Cond
	
		appendCond    []*sync.Cond 
	
		state     RaftState
	
		electionTime  *time.Timer
		heartBeatTime *time.Timer
	
		// // persistant state on all servers, updated on storage before responding to rpcs
		currentTerm    int
		votedFor       int
		log            []Entry          
	
		// // volatile state on all servers
		commitIndex    int
		lastApplied    int
	
		// // volatile state on the leader, reinitialized after each election
		nextIndex      []int
		matchIndex     []int
	
	}
	
以下具体说说代码思路

### make

make方法是创建一个新的raft节点。

- 对相关字段进行初始化

- 调用readPersist恢复persistant state

- 创建三类协程，timer负责开启投票和心跳包；appendAndWait负责将log append到其他raft节点；applyLogs负责将可应用的log应用到状态机，没有应用的时候则wait

首先在此对三类协程进行解析，这里的思路主要是参考了https://github.com/OneSizeFitsQuorum/MIT6.824-2021/blob/master/docs/lab2.md

参考这位大佬的思路有利于我对多线程编程的理解，这一直是我的漏洞。通过channel、条件变量、goruntime这三个的使用，大大加深我对go多线程编程的理解。


### timer（选举相关和发送log相关）

timer函数使用select，一旦定时器超时，通过channel通知timer函数，这是go的特色

+ 若是选举计时器超时并且当前状态不是leader，变为candidate，term加1，开启选举KickOffElection()，该函数创建相关协程处理选举后马上返回，返回后刷新选举时间

+ 若是心跳包超时并且当前状态是leader，发送心跳包boardcast()，该函数创建相关协程或者唤醒appendAndWait后马上返回，返回后刷新心跳时间

### appendAndWait（发送log 相关）

该协程每个peer(除了自己)都有一个对应，负责尽全力将新的log append到对应的peer中去。

+ 循环调用needAppend判断是否需要发送log，不需要我们则wait

+ 需要的话，我们调用append2peer将log发送给对应peer，该函数确保正确处理一切发送的结果，随后返回，继续上述循环

### applyLogs（应用相关）

该协程负责尽全力将log应用到状态机中，也就是写入applyCh

+ 首先循环判断是否需要将log应用到状态机，不需要我们则wait

+ 需要的话，首先将需要apply的log拷贝出来，然后解锁，将log写入applyCh，因为写入applyCh是一件很耗时的事情，不应该拿着锁

+ 最后修改lastApplied即可，注意这里修改的时候需要判断rf.lastApplied和旧的commitIndex大小，详情见代码


### 选举相关：

结构体和论文上的一致

##### KickOffElection

发起选举，由timer函数负责调用。该函数需要立即返回，所以为所有peer开启一个发送和处理选举的协程，之后马上返回

+ 设立协程之间的共享变量votecnt，负责统计赞成的次数

+ 为所有peer准备发送的参数

+ 为所有peer开启一个发送和处理选举的协程
	
	- 协程首先调用sendRequestVote
	
	- 收到返回后再获取锁
	
	- 若是赞成票，将votecnt加1，判断是否过半，过半则变为leader，调用boardcast广播，该函数必须立即广播，然后刷新心跳时间
	
	- 若回复的term比currentterm大，我们则变为follow

+ 返回

##### RequestVote

被candidate远程调用，负责处理candidate发送的投票请求

+ 若参数的term比currentterm小，直接返回false，并且将自己的term附加上去

+ 若currentterm比参数的term小，变为follow，并且将votefor清空，刷新选举时间

+ 最后判断是否满足投票条件（见论文），满足我们则将票投给他，不满足则不投给他


### 发送log相关：

结构体和论文上的几乎一致，为加速回退，返回值新建ConflictTermIdx表示发生冲突周期最小的index

##### start

接收客户端的请求，只有leader才能对其进行处理，不是leader直接返回-1和false

+ 首先判断是否是leader，不是则直接返回-1和false

+ 否则我们将对应的log加上去，调用boardcast广播log到其他raft服务器需要马上返回

+ 正确返回参数

##### boardcast

被timer、start和KickOffElection函数调用，负责广播log，需要立即返回

+ 若是KickOffElection和timer这两个函数调用，需要立即发送心跳包，则马上创建append2peer协程发送，之后立即返回

+ 否则调用signal唤醒所有peer对应的appendAndWait协程，让它们发送

##### append2peer

全权处理一次发送log给对应的peer，除非返回的term比currentterm大或者崩溃、被杀死，否则一定成功将对应log发送

+ 首先获取锁，调用makeAppendEntriesArgs新建发送的args

+ 进行一次判断是否是leader，不是我们则直接返回，不能发送。注意每次我们重新获取锁之后我们都要进行对应状态判断（所有方法都是一样，只能做对应状态允许做的事情）

+ 不是我们则直接释放锁，调用sendAppendEntries发送

+ 收到返回值后我们获取锁，进行判断，若不是leader，我们直接放锁返回

+ 若返回的term比currentterm小，丢弃不处理

+ 若返回的term比currentterm大，更改currentterm，修改state为follower，清空votefor，放锁返回

+ 否则，我们判断append是否成功，成功我们则重置对应的nextIndex和matchIndex，调用leaderCommit对commit相关事务处理，该函数必须要马上返回，返回后结束函数

+ 若不成功，我们使用返回的ConflictTermIdx重置对应的nextIndex，重新调用makeAppendEntriesArgs新建args回到第三步循环发送

##### AppendEntries

+ 若参数的term比currentterm小，直接返回false，并且将自己的term附加上去

+ 若currentterm比参数的term小，变为follow，并且将votefor清空，刷新选举时间

+ 调用checkAppendCondition判断是否能加上log，能我们则直接加上去，不能我们则计算ConflictTermIdx，将其返回之

+ 若能加上log之后，如果LeaderCommit > rf.commitIndex，我们调用followerCommit应用log



### 应用log相关：

##### leaderCommit

判断是否需要应用log，如果需要，我们更新commitIndex，调用signal唤醒applyLogs

##### followerCommit

更新commitIndex，调用signal唤醒applyLogs


### 持久化

再修改持久状态时我们调用persist存储即可



























