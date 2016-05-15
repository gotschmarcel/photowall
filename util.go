// Copyright 2016 Marcel Gotsch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

// ceilIntDivision divides a by b and ceils the result
//
// Note: This function works only for positive non-zero
// numbers
func ceilIntDivision(a, b int) int {
	return 1 + ((a - 1) / b)
}
