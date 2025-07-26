package data

type Patient struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Lastname string `json:"lastname"`
}

type Medic struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Lastname string `json:"lastname"`
}

type Vaccine struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type VaccinationRecord struct {
	Date    string  `json:"date"`
	Patient Patient `json:"patient"`
	Medic   Medic   `json:"medic"`
	Vaccine Vaccine `json:"vaccine"`
}

type VaccinationResponse struct {
	Vaccinations  []VaccinationRecord `json:"vaccinations"`
	VaccineCounts map[string]int      `json:"vaccineCounts"`
}
