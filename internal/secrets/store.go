package secrets

type RealitySecret struct {
	PrivateKey string   `json:"private_key"`
	PublicKey  string   `json:"public_key"`
	ShortIDs   []string `json:"short_ids"`
	CreatedAt  int64    `json:"created_at"`
}

type Store interface {
	LoadToken() (string, error)
	SaveToken(token string) error
	LoadReality() (RealitySecret, error)
	SaveReality(secret RealitySecret) error
}
