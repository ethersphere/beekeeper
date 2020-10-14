// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bee

import (
	bmtlegacy "github.com/ethersphere/bmt/legacy"
	"github.com/ethersphere/bmt/pool"
)

var instance pool.Pooler

func init() {
	instance = pool.New(8, 128)
}

// Get a bmt Hasher instance.
// Instances are reset before being returned to the caller.
func GetBmt() *bmtlegacy.Hasher {
	return instance.Get()
}

// Put a bmt Hasher back into the pool
func PutBmt(h *bmtlegacy.Hasher) {
	instance.Put(h)
}
