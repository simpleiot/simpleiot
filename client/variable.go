package client

// Variable represents the config of a variable node type
type Variable struct {
	ID           string             `node:"id"`
	Parent       string             `node:"parent"`
	Description  string             `point:"description"`
	VariableType string             `point:"variableType"`
	Value        map[string]float64 `point:"value"`
}
