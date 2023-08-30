package models

type TaskBuilder struct {
	// Task is the task to be built
	task *Task
}

func NewTaskBuilder() *TaskBuilder {
	return &TaskBuilder{
		task: &Task{},
	}
}

func NewTaskBuilderFromTask(task *Task) *TaskBuilder {
	return &TaskBuilder{
		task: task,
	}
}

func (b *TaskBuilder) Name(name string) *TaskBuilder {
	b.task.Name = name
	return b
}

func (b *TaskBuilder) Engine(engine *SpecConfig) *TaskBuilder {
	b.task.Engine = engine
	return b
}

func (b *TaskBuilder) Publisher(publisher *SpecConfig) *TaskBuilder {
	b.task.Publisher = publisher
	return b
}

func (b *TaskBuilder) ResourcesConfig(resourcesConfig *ResourcesConfig) *TaskBuilder {
	b.task.ResourcesConfig = resourcesConfig
	return b
}

func (b *TaskBuilder) InputSources(inputSources ...*InputSource) *TaskBuilder {
	b.task.InputSources = inputSources
	return b
}

func (b *TaskBuilder) ResultPaths(resultPaths ...*ResultPath) *TaskBuilder {
	b.task.ResultPaths = resultPaths
	return b
}

func (b *TaskBuilder) Network(network *NetworkConfig) *TaskBuilder {
	b.task.Network = network
	return b
}

func (b *TaskBuilder) Timeouts(timeouts *TimeoutConfig) *TaskBuilder {
	b.task.Timeouts = timeouts
	return b
}

func (b *TaskBuilder) RestartPolicy(policy RestartPolicyType) *TaskBuilder {
	b.task.RestartPolicy = policy
	return b
}

func (b *TaskBuilder) Build() (*Task, error) {
	b.task.Normalize()
	return b.task, b.task.Validate()
}

// BuildOrDie is the same as Build, but panics if an error occurs
func (b *TaskBuilder) BuildOrDie() *Task {
	task, err := b.Build()
	if err != nil {
		panic(err)
	}
	return task
}
