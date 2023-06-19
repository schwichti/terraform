// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package moduletest

type File struct {
	Name   string
	Status Status

	Runs []*Run
}
