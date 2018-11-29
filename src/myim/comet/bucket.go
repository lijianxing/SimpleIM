package main

import (
	"myim/libs/proto"
	"sync"
)

type BucketOptions struct {
	ChannelSize int
}

// Bucket is a channel holder.
type Bucket struct {
	cLock    sync.RWMutex        // protect the channels for chs
	chs      map[string]*Channel // map sub key to a channel
	boptions BucketOptions
}

// NewBucket new a bucket struct. store the key with im channel.
func NewBucket(boptions BucketOptions) (b *Bucket) {
	b = new(Bucket)
	b.chs = make(map[string]*Channel, boptions.ChannelSize)
	b.boptions = boptions
	return
}

// ChannelCount channel count in the bucket
func (b *Bucket) ChannelCount() int {
	// 线程安全么?
	return len(b.chs)
}

// Put put a channel according with sub key.
func (b *Bucket) Put(key string, ch *Channel) (err error) {
	b.cLock.Lock()
	b.chs[key] = ch
	b.cLock.Unlock()
	return
}

// Del delete the channel by sub key.
func (b *Bucket) Del(key string) {
	b.cLock.Lock()
	if _, ok := b.chs[key]; ok {
		delete(b.chs, key)
	}
	b.cLock.Unlock()
}

// Channel get a channel by sub key.
func (b *Bucket) Channel(key string) (ch *Channel) {
	b.cLock.RLock()
	ch = b.chs[key]
	b.cLock.RUnlock()
	return
}

// 全量推送
// Broadcast push msgs to all channels in the bucket.
func (b *Bucket) Broadcast(p *proto.Proto) {
	var ch *Channel
	b.cLock.RLock()
	for _, ch = range b.chs {
		// ignore error
		ch.Push(p)
	}
	b.cLock.RUnlock()
}
