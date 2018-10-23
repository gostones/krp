// Copyright 2017 The Goboot Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//based on https://github.com/cenkalti/backoff/blob/master/exponential.go
package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

type Operation func() error

func Retry(op Operation, bo ...BackOff) (err error) {
	var b BackOff
	if len(bo) == 0 {
		b = NewDefaultBackOff()
	} else {
		b = bo[0]
	}
	for i := 0; i < b.attempts; i++ {
		log.Printf("Retry count:  %d\n", i)

		if err = op(); err == nil {
			return nil
		}

		d := b.randomValue()

		log.Printf("Operation error:  %s, will retry after %s\n", err, d)

		time.Sleep(d)

		log.Printf("Retrying after %s ...\n", d)
	}
	return fmt.Errorf("Failed after %d attempts, last error: %s", b.attempts, err)
}

// exponential backoff
type BackOff struct {
	attempts int
	interval time.Duration
}

// contruct a backoff with provided attempts and duration
func NewBackOff(attempts int, d time.Duration) BackOff {
	return BackOff{attempts: attempts, interval: d}
}

// construct a backoff with default values
func NewDefaultBackOff() BackOff {
	return NewBackOff(defaultAttempts, defaultInterval)
}

const (
	defaultInterval = 1000 * time.Millisecond
	defaultAttempts = 3

	randomizationFactor = 0.5
	multiplier          = 2.0
)

// Returns a random value from range: [randomizationFactor * interval, randomizationFactor * interval].
func (b *BackOff) randomValue() time.Duration {
	i := b.interval
	b.interval = time.Duration(float64(i) * multiplier)

	r := rand.Float64()
	var delta = randomizationFactor * float64(i)
	var min = float64(i) - delta
	var max = float64(i) + delta

	return time.Duration(min + (r * (max - min + 1)))
}
