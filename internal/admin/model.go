package admin


type Admin struct {
	ID int `json:"id"`
	Email string `json:"email"`
	Senha string `json:"senha"`
	Cidade string `json:"cidade"`
}

type AdminInput struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
	Cidade string `json:"cidade"`
}