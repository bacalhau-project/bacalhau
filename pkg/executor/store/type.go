package store

type ExecutorStore interface {
	Get(id string) (interface{}, bool)
	Set(id string, data interface{})
	Delete(id string)
	List() map[string]interface{}
}
