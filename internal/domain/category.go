package domain

type Category string

const (
	CategoryResistor           Category = "resistor"
	CategoryCapacitor          Category = "capacitor"
	CategoryInductor           Category = "inductor"
	CategoryIntegratedCircuit  Category = "integrated_circuit"
	CategoryFerriteBead        Category = "ferrite_bead"
	CategoryDiode              Category = "diode"
	CategoryLED                Category = "led"
	CategoryTransistorBJT      Category = "transistor_bjt"
	CategoryTransistorMOSFET   Category = "transistor_mosfet"
	CategoryRegulatorLinear    Category = "regulator_linear"
	CategoryRegulatorSwitching Category = "regulator_switching"
	CategoryConnector          Category = "connector"
	CategorySwitch             Category = "switch"
	CategoryCrystalOscillator  Category = "crystal_oscillator"
	CategoryFuse               Category = "fuse"
	CategoryBattery            Category = "battery"
	CategorySensor             Category = "sensor"
	CategoryModule             Category = "module"
)

type Operator string

const (
	OperatorEqual Operator = "eq"
	OperatorGTE   Operator = "gte"
	OperatorLTE   Operator = "lte"
)
