package check

type DummyCheckExecutor struct {
	Started bool
}

func (d *DummyCheckExecutor) Start() {
	d.Started = true
}

func (d *DummyCheckExecutor) Failed() bool {
	return false
}

func (d *DummyCheckExecutor) LastError() error {
	return nil
}
