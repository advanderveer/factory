# factory

## A Serverless Scheduling Component


KeepNode() :inc node ttl, return expired if node is removed or expired
RegisterNode(size:1) :put a new node with the given capacity
DeregisterNode() :remove the node, task/node heartbeats will return expire

ScheduleTask(size:5) :add a task to the scheduling queue
ReceiveTasks() :receive task handles
KeepTask()
FailTask()
CompleteTask()



<!-- node.keep() -> inc node ttl, return expired if expired
node.register(cap: 10) -> add node, return task iterator
node.deregister() -> remove node, proc/node (heartbeats) will fail

proc.keep() -> inc proc ttl, return expired if expired

schedule() -> add a proc to the scheduling queue
receive() -> return a stream of proc handlers

handle.keep() -> inc proc ttl, keeps the handle valid
handle.failed() -> proc ended unexpectedly, report what happended
handle.success() -> proc ended gracefully, report  -->
