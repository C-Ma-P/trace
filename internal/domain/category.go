package domain

type Category string

const (
	CategoryResistor          Category = "resistor"
	CategoryCapacitor         Category = "capacitor"
	CategoryInductor          Category = "inductor"
	CategoryIntegratedCircuit Category = "integrated_circuit"
)

type Operator string

const (
	OperatorEqual Operator = "eq"
	OperatorGTE   Operator = "gte"
	OperatorLTE   Operator = "lte"
)
