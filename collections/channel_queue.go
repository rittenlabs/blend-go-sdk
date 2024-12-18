/*

Copyright (c) 2022 - Present. Blend Labs, Inc. All rights reserved
Use of this source code is governed by a MIT license that can be found in the LICENSE file.

*/

package collections

import "sync"

// NewChannelQueueWithCapacity returns a new ChannelQueue instance.
func NewChannelQueueWithCapacity(capacity int) *ChannelQueue {
	return &ChannelQueue{Capacity: capacity, storage: make(chan interface{}, capacity), latch: sync.Mutex{}}
}

// NewChannelQueueFromValues returns a new ChannelQueue from a given slice of values.
func NewChannelQueueFromValues(values []interface{}) *ChannelQueue {
	capacity := len(values)
	cq := &ChannelQueue{Capacity: capacity, storage: make(chan interface{}, capacity), latch: sync.Mutex{}}
	for _, v := range values {
		cq.storage <- v
	}
	return cq
}

// ChannelQueue is a threadsafe queue.
type ChannelQueue struct {
	Capacity int
	storage  chan interface{}
	latch    sync.Mutex
}

// Len returns the number of items in the queue.
func (cq *ChannelQueue) Len() int {
	return len(cq.storage)
}

// Enqueue adds an item to the queue.
func (cq *ChannelQueue) Enqueue(item interface{}) {
	cq.storage <- item
}

// Dequeue returns the next element in the queue.
func (cq *ChannelQueue) Dequeue() interface{} {
	if len(cq.storage) != 0 {
		return <-cq.storage
	}
	return nil
}

// DequeueBack iterates over the queue, removing the last element and returning it
func (cq *ChannelQueue) DequeueBack() interface{} {
	values := []interface{}{}
	storageLen := len(cq.storage)
	for x := 0; x < storageLen; x++ {
		v := <-cq.storage
		values = append(values, v)
	}
	var output interface{}
	for index, v := range values {
		if index == len(values)-1 {
			output = v
		} else {
			cq.storage <- v
		}
	}
	return output
}

// Peek returns (but does not remove) the first element of the queue.
func (cq *ChannelQueue) Peek() interface{} {
	if len(cq.storage) == 0 {
		return nil
	}
	return cq.Contents()[0]
}

// PeekBack returns (but does not remove) the last element of the queue.
func (cq *ChannelQueue) PeekBack() interface{} {
	if len(cq.storage) == 0 {
		return nil
	}
	return cq.Contents()[len(cq.storage)-1]
}

// Clear clears the queue.
func (cq *ChannelQueue) Clear() {
	cq.storage = make(chan interface{}, cq.Capacity)
}

// Each pulls every value out of the channel, calls consumer on it, and puts it back.
func (cq *ChannelQueue) Each(consumer func(value interface{})) {
	if len(cq.storage) == 0 {
		return
	}
	values := []interface{}{}
	for len(cq.storage) != 0 {
		v := <-cq.storage
		consumer(v)
		values = append(values, v)
	}
	for _, v := range values {
		cq.storage <- v
	}
}

// Consume pulls every value out of the channel, calls consumer on it, effectively clearing the queue.
func (cq *ChannelQueue) Consume(consumer func(value interface{})) {
	if len(cq.storage) == 0 {
		return
	}
	for len(cq.storage) != 0 {
		v := <-cq.storage
		consumer(v)
	}
}

// EachUntil pulls every value out of the channel, calls consumer on it, and puts it back and can abort mid process.
func (cq *ChannelQueue) EachUntil(consumer func(value interface{}) bool) {
	contents := cq.Contents()
	for x := 0; x < len(contents); x++ {
		if consumer(contents[x]) {
			return
		}
	}
}

// ReverseEachUntil pulls every value out of the channel, calls consumer on it, and puts it back and can abort mid process.
func (cq *ChannelQueue) ReverseEachUntil(consumer func(value interface{}) bool) {
	contents := cq.Contents()
	for x := len(contents) - 1; x >= 0; x-- {
		if consumer(contents[x]) {
			return
		}
	}
}

// Contents iterates over the queue and returns an array of its contents.
func (cq *ChannelQueue) Contents() []interface{} {
	values := []interface{}{}
	storageLen := len(cq.storage)
	for x := 0; x < storageLen; x++ {
		v := <-cq.storage
		values = append(values, v)
	}
	for _, v := range values {
		cq.storage <- v
	}
	return values
}

// Drain iterates over the queue and returns an array of its contents, leaving it empty.
func (cq *ChannelQueue) Drain() []interface{} {
	values := []interface{}{}
	for len(cq.storage) != 0 {
		v := <-cq.storage
		values = append(values, v)
	}
	return values
}
