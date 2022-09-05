# MIT6.824
MIT2020年秋分布式系统实验

## lab1

实验一是在单机上通过多线程实现mapreduce，对于我这种刚开始学分布式和从来没有接触过go的同学一开始会比较痛苦。首先我花了两小时去看824的第一节课，然后花了大概一小时把mapreduce
论文看了个大概，最后花了两个小时入门了go（没有入门goruntime和channel）然后在阅读实验一指导书和慢慢写代码的时候入门了一下channel和goruntime语法。然后渐渐地理清了思路。在写
实验之前我先把实验一的思路写下

### worker

work的实现较为简单

- 通过rpc请求master分配相关任务，将请求任务的信息填入Req结构体

- 根据Job中的信息来判断任务的类型，任务分为map，reduce，wait和kill

- map则调用map处理函数

  + 打开job中传入的文件数组中所有文件（只有一个）
  
  + 读取所有kv数据
  
  + 调用map函数处理kv，输出到intermediate
  
  + 对intermediate进行排序（在实际的海量数据中，应该使用的是外部排序算法）
  
  + 准备内存哈希表，调用ihash(kv.Key) % job.nreduce计算相应kv应该加入的bucket（在实际的海量数据中，该哈希表应该是放在磁盘）
  
  + 按实验提示要求调用Encode存入相关文件，注意文件命名规范

  + 调用call函数告知master任务完成

- reduce则调用reduce处理函数

  + 按实验提示Decode所有通过job传入的存储中间kv的文件

  + 将kv排序

  + 按taskid创建文件，调用reduce函数处理，将结果写入文件

  + 调用call函数告知master任务完成

- wait则等待2s

- kill则结束当前work

- 循环请求直到被传入kill


### master

master的实现则是核心

#### HandlerCall

#### HandlerCrash
















