package core

import "log"

type Queue struct {
	maxSize  int
	front    int
	rear     int
	elements []interface{}
}

func NewQueue(max_size int) *Queue {
	queue := new(Queue)
	queue.front = 0
	queue.rear = 0
	queue.maxSize = max_size
	queue.elements = make([]interface{}, max_size)

	return queue
}

func (queue *Queue) Enqueue(element interface{}) {
	new_rear := (queue.rear + 1) % queue.maxSize
	if new_rear == queue.front {
		log.Fatal("Error: Queue is full")
		return
	}
	queue.rear = new_rear
	queue.elements[new_rear] = element
}

func (queue *Queue) Dequeue() interface{} {
	if queue.front == queue.rear {
		//queue is empty
		return nil
	}

	queue.front = (queue.front + 1) % queue.maxSize
	return queue.elements[queue.front]
}

func (queue *Queue) IsEmpty() bool {
	return queue.front == queue.rear
}
