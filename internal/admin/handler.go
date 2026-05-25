package admin

type AdminHandler struct {
	s AdminStore
}

type CreateAdminRequest struct {
	Email  string `json:"email"`
	Senha  string `json:"senha"`
	Cidade string `json:"cidade"`
}

type CreateAdminResponse struct {
	ID int `json:"id"`
}

type AdminResponse struct {
	ID     int    `json:"id"`
	Email  string `json:"email"`
	Cidade string `json:"cidade"`
}

func NewAdminHandler(store AdminStore) *AdminHandler {
	return &AdminHandler{s: store}
}
