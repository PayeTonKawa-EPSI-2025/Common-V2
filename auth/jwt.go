package auth

type Claims struct {
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`

	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`

	Roles []string `json:"roles"`
}

// On normalise pour toujours utiliser claims.Roles
func (c *Claims) Normalize() {
	if len(c.Roles) == 0 {
		c.Roles = c.RealmAccess.Roles
	}
}
