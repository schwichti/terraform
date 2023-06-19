// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package moduletest

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Run struct {
	Name   string
	Status Status

	Diagnostics tfdiags.Diagnostics
}
