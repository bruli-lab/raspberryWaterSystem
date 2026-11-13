package program

type Execution struct {
	seconds Seconds
	zones   []string
}

func (e *Execution) Seconds() Seconds {
	return e.seconds
}

func (e *Execution) Zones() []string {
	return e.zones
}

func (e *Execution) validate() error {
	if _, err := ParseSeconds(e.seconds.Int()); err != nil {
		return err
	}
	if len(e.zones) == 0 {
		return ErrEmptyExecutionZones
	}
	return nil
}

func NewExecution(seconds Seconds, zones []string) (Execution, error) {
	ex := Execution{seconds: seconds, zones: zones}
	if err := ex.validate(); err != nil {
		return Execution{}, err
	}
	return ex, nil
}

func HydrateExecution(seconds Seconds, zones []string) Execution {
	return Execution{seconds: seconds, zones: zones}
}
