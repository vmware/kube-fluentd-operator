package controller

import (
	"context"
	"time"
)

// Updater provides a means to notify through a channel that an update is necessary
type Updater interface {
	GetUpdateChannel() <-chan time.Time
}

// FixedTimeUpdater is an Updater that delivers a notification after a fixed amount of time
type FixedTimeUpdater struct {
	interval time.Duration
}

var _ Updater = &FixedTimeUpdater{}

func NewFixedTimeUpdater(ctx context.Context, seconds int) *FixedTimeUpdater {
	return &FixedTimeUpdater{interval: time.Duration(seconds) * time.Second}
}

func (f *FixedTimeUpdater) GetUpdateChannel() <-chan time.Time {
	return time.After(f.interval)
}

// OnDemandUpdater is an Updater that delivers notifications on demand through a shared channel
type OnDemandUpdater struct {
	channel chan time.Time
}

var _ Updater = &OnDemandUpdater{}

func NewOnDemandUpdater(ctx context.Context, channel chan time.Time) *OnDemandUpdater {
	return &OnDemandUpdater{channel: channel}
}

func (o *OnDemandUpdater) GetUpdateChannel() <-chan time.Time {
	return o.channel
}
