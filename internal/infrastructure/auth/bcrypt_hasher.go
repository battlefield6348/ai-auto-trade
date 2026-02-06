package authinfra

import "golang.org/x/crypto/bcrypt"

// BcryptHasher 使用 bcrypt 檢查密碼。
type BcryptHasher struct{}

func (BcryptHasher) Compare(hashed, plain string) bool {
	if hashed == "" || plain == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}

func (BcryptHasher) Hash(plain string) (string, error) {
	return HashPassword(plain)
}


// HashPassword 供 seed 使用，產生 bcrypt 雜湊。
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
