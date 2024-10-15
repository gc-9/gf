package auth

import "golang.org/x/crypto/bcrypt"

func GenerateFromPassword(pass string) string {
	buf, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	return string(buf)
}

func CompareHashAndPassword(hashedPassword, pass string) bool {
	return bcrypt.CompareHashAndPassword(
		[]byte(hashedPassword),
		[]byte(pass)) == nil
}
