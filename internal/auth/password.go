package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword returns a bcrypt hash of plain. Cost is taken from the caller's
// Config so tests can use bcrypt.MinCost without slowing the suite.
func HashPassword(plain string, cost int) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword reports whether plain matches the stored bcrypt hash.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
