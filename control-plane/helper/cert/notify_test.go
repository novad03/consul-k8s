// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cert

import (
	"context"
	"testing"
	"time"
)

func TestNotify(t *testing.T) {
	t.Parallel()

	// Source is just randomly generated
	source := testGenSource()
	source.Expiry = 5 * time.Second
	source.ExpiryWithin = 2 * time.Second

	// Create notifier
	ch := make(chan MetaBundle)
	n := &Notify{Ch: ch, Source: source}
	defer n.Stop()
	go n.Start(context.Background())

	// We should receive an update almost immediately
	select {
	case <-time.After(250 * time.Millisecond):
		t.Fatal("should've received initial bundle")
	case b := <-ch:
		testBundleVerify(t, &b.Bundle)
	}

	// We should not receive an update for at least one second
	select {
	case <-time.After(750 * time.Millisecond):
	case <-ch:
		t.Fatal("should not receive update")
	}

	b := <-ch
	testBundleVerify(t, &b.Bundle)
}
