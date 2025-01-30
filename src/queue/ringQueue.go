/*
This package provides a simple implementation of a queue data structure.

The queue is implemented as a circular doubly-linked list. Although the last element's "Next" pointer is set to nil, the Next() function seamlessly transitions from the last element back to the head of the queue. This design enables efficient and continuous iteration over the elements in the queue.

The queue implemented here is as fast as it is for an additional reason: it is *not* thread-safe.
*/
package queue

type QueueElement[T any] struct {
	Value T
	Next *QueueElement[T]
	Previous *QueueElement[T]
}

// Queue represents a single instance of the queue data structure.
type RingQueue[T any] struct {
	head *QueueElement[T]
	current *QueueElement[T]
	count int
}


// New constructs and returns a new Queue.
func New[T any]() *RingQueue[T] {
	return &RingQueue[T]{}
}

// Length returns the number of elements currently stored in the queue.
func (q *RingQueue[T]) Length() int {
	return q.count
}

// Current returns the current value of the queue
func (q *RingQueue[T]) Current() T {
    return q.current.Value
}

// Head returns the head value of the queue
func (q *RingQueue[T]) Head() T {
	return q.head.Value
}

// HeadElement returns the head element of the queue
func (q *RingQueue[T]) HeadElement() *QueueElement[T] {
	return q.head
}

// Return the current value of the queue and move the pointer to the next element
func (q *RingQueue[T]) Next() T {
	current := q.current

	if q.current.Next == nil {
		q.current = q.head
	} else {
		q.current = q.current.Next
	}
	
	return current.Value
}


// Add an element to the queue. If the queue is empty, the element will be both the head and the current element.
func (q *RingQueue[T]) Add(elem T) {
	newElement := &QueueElement[T]{Value: elem}
	if q.Length() == 0 {
		q.head = newElement
		q.current = newElement
    } else {
		newElement.Next = q.head
		q.head.Previous = newElement
		q.head = newElement
	}
	q.count++
}


// Removes current element, sets the current element to the next, and returns the deleted element. If the
// queue is empty, the call will panic.
func (q *RingQueue[T]) Remove() T {
	if q.head == nil {
		panic("Remove called on empty queue")
	}

	current := q.current

	if q.Length() == 1 {
		q.head = nil
		q.current = nil
		q.count = 0
		return current.Value
	}

	if current.Next == nil {
		q.current = q.head
	} else {
		previous := current.Previous

		if previous!= nil {
			// Removing element from the chain
			previous.Next = current.Next

			if q.current.Next != nil {
				q.current = q.current.Next
			} else {
				q.current = q.head
			}
        } else {
			// Removing head element
            q.head = q.head.Next
			q.head.Previous = nil
			q.current = q.head
        }
	}
	
	q.count--
	return current.Value
}
