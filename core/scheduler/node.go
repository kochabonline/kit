package scheduler

import (
	"context"
	"time"

	"github.com/kochabonline/kit/log"
)

const (
	executing = "executing"
	failed    = "failed"
	completed = "completed"
)

type Node struct {
	Id        string
	hashRing  *ConsistentHash
	taskQueue []Task
	taskStore map[string]string
	lock      *DistributedLock
}

func NewNode(id string, lock *DistributedLock, taskQueue []Task) *Node {
	return &Node{
		Id:        id,
		hashRing:  NewConsistentHash(3),
		taskStore: make(map[string]string),
		lock:      lock,
		taskQueue: taskQueue,
	}
}

func (n *Node) ScheduleTasks() {
	for _, task := range n.taskQueue {
		assignedNode := n.hashRing.Get(task.Id)
		if assignedNode != n.Id {
			continue // Skip if the task is not assigned to this node
		}

		// Lock the task
		lockKey := "lock:" + task.Id
		lockValue := n.Id
		ctx := context.Background()

		if !n.lock.TryLock(ctx, lockKey, lockValue, 10*time.Second) {
			continue // Skip if the task is being executed by another node
		}

		// Process the task
		n.taskStore[task.Id] = executing

		// TODO: 执行任务
		log.Infof("excutor: %v", task.Excutor)
		time.Sleep(3 * time.Second)
		n.taskStore[task.Id] = completed

		// Unlock the task
		n.lock.Unlock(ctx, lockKey, lockValue)
	}
}

func (n *Node) ScheduleTask(taskId string) {
	assignedNode := n.hashRing.Get(taskId)
	if assignedNode != n.Id {
		log.Infof("Task %s assigned to node %s", taskId, assignedNode)
		return
	}

	// Lock the task
	lockKey := "lock:" + taskId
	lockValue := n.Id
	ctx := context.Background()

	if !n.lock.TryLock(ctx, lockKey, lockValue, 10*time.Second) {
		log.Infof("Task %s is being executed by another node", taskId)
		return
	}

	// TODO: 执行任务
	log.Info("执行任务")

	// Process the task
	log.Infof("Task %s is being executing by node %s", taskId, n.Id)
	n.taskStore[taskId] = executing

	// Unlock the task
	n.taskStore[taskId] = completed
	n.lock.Unlock(ctx, lockKey, lockValue)
	log.Infof("Task %s is completed by node %s", taskId, n.Id)
}

func (n *Node) HandleFailure(failedNode string) {
	log.Infof("Node %s detected failure of %s, reassigning tasks...", n.Id, failedNode)

	for taskId, status := range n.taskStore {
		if status != completed {
			assignedNode := n.hashRing.Get(taskId)
			if assignedNode == n.Id {
				n.ScheduleTask(taskId)
			}
		}
	}
}
