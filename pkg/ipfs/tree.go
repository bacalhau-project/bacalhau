package ipfs

import (
	"context"

	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
)

type IPLDTreeNode struct {
	Cid      cid.Cid
	Path     []string
	Children []IPLDTreeNode
}

func getTreeNode(ctx context.Context, navNode ipld.NavigableNode, path []string) (IPLDTreeNode, error) {
	var children []IPLDTreeNode
	for i, link := range navNode.GetIPLDNode().Links() {
		childNavNode, err := navNode.FetchChild(ctx, uint(i))
		if err != nil {
			return IPLDTreeNode{}, err
		}
		childTreeNode, err := getTreeNode(ctx, childNavNode, append(path, link.Name))
		if err != nil {
			return IPLDTreeNode{}, err
		}
		children = append(children, childTreeNode)
	}
	node := IPLDTreeNode{
		Cid:      navNode.GetIPLDNode().Cid(),
		Path:     path,
		Children: children,
	}
	return node, nil
}

func FlattenTreeNode(ctx context.Context, rootNode IPLDTreeNode) ([]IPLDTreeNode, error) {
	nodes := []IPLDTreeNode{rootNode}

	for _, child := range rootNode.Children {
		nodeChildren, err := FlattenTreeNode(ctx, child)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, nodeChildren...)
	}

	return nodes, nil
}
