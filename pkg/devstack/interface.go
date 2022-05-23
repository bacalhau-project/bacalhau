package devstack

type IDevStack interface {
	AddTextToNodes(nodeCount int, fileContent []byte) (string, error)
	AddFileToNodes(nodeCount int, filePath string) (string, error)
}
