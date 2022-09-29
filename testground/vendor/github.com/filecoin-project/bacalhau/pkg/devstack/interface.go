package devstack

import "context"

type IDevStack interface {
	AddTextToNodes(ctx context.Context, nodeCount int, fileContent []byte) (string, error)
	AddFileToNodes(ctx context.Context, nodeCount int, filePath string) (string, error)
}
