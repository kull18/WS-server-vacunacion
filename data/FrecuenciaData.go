package data

type FrecuenciaData struct {
	Intervalos []string  `json:"intervalos"`
	Marcas     []float64 `json:"marcas"`
	Fa         []int     `json:"fa"`
	Fr         []float64 `json:"fr"`
	Fac        []int     `json:"fac"`
}