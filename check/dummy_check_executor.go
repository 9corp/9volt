package check

type DummyCheckExecutor struct {
	Started bool
	Stopped bool
}

func (d *DummyCheckExecutor) Start() error {
	d.Started = true
	return nil
}

func (d *DummyCheckExecutor) Stop() error {
	d.Stopped = true
	return nil
}
