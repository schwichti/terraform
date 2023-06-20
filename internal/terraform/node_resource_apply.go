// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodeExpandApplyableResource handles the first layer of resource
// expansion during apply. Even though the resource instances themselves are
// already expanded from the plan, we still need to expand the
// NodeApplyableResource nodes into their respective modules.
type nodeExpandApplyableResource struct {
	*NodeAbstractResource
}

var (
	_ GraphNodeDynamicExpandable    = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeReferenceable        = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeReferencer           = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeConfigResource       = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeAttachResourceConfig = (*nodeExpandApplyableResource)(nil)
	_ graphNodeExpandsInstances     = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeTargetable           = (*nodeExpandApplyableResource)(nil)
)

func (n *nodeExpandApplyableResource) expandsInstances() {
}

func (n *nodeExpandApplyableResource) References() []*addrs.Reference {
	refs := n.NodeAbstractResource.References()

	// The expand node needs to connect to the individual resource instances it
	// references, but cannot refer to it's own instances without causing
	// cycles. It would be preferable to entirely disallow self references
	// without the `self` identifier, but those were allowed in provisioners
	// for compatibility with legacy configuration. We also can't always just
	// filter them out for all resource node types, because the only method we
	// have for catching certain invalid configurations are the cycles that
	// result from these inter-instance references.
	return filterSelfRefs(n.Addr.Resource, refs)
}

func (n *nodeExpandApplyableResource) Name() string {
	return n.NodeAbstractResource.Name() + " (expand)"
}

func (n *nodeExpandApplyableResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var g Graph

	expander := ctx.InstanceExpander()
	moduleInstances := expander.ExpandModule(n.Addr.Module)
	for _, module := range moduleInstances {
		g.Add(&NodeApplyableResource{
			NodeAbstractResource: n.NodeAbstractResource,
			Addr:                 n.Addr.Resource.Absolute(module),
		})
	}
	addRootNodeToGraph(&g)

	return &g, nil
}

// NodeApplyableResource represents a resource that is "applyable":
// it may need to have its record in the state adjusted to match configuration.
//
// Unlike in the plan walk, this resource node does not DynamicExpand. Instead,
// it should be inserted into the same graph as any instances of the nodes
// with dependency edges ensuring that the resource is evaluated before any
// of its instances, which will turn ensure that the whole-resource record
// in the state is suitably prepared to receive any updates to instances.
type NodeApplyableResource struct {
	*NodeAbstractResource

	Addr addrs.AbsResource
}

var (
	_ GraphNodeModuleInstance       = (*NodeApplyableResource)(nil)
	_ GraphNodeConfigResource       = (*NodeApplyableResource)(nil)
	_ GraphNodeExecutable           = (*NodeApplyableResource)(nil)
	_ GraphNodeProviderConsumer     = (*NodeApplyableResource)(nil)
	_ GraphNodeAttachResourceConfig = (*NodeApplyableResource)(nil)
	_ GraphNodeReferencer           = (*NodeApplyableResource)(nil)
)

func (n *NodeApplyableResource) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// GraphNodeExecutable
func (n *NodeApplyableResource) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	if n.Config == nil {
		// Nothing to do, then.
		log.Printf("[TRACE] NodeApplyableResource: no configuration present for %s", n.Name())
		return nil
	}

	return n.writeResourceState(ctx, n.Addr)
}
