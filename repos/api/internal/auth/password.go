package auth

import (
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	passwordMinRunes      = 10
	passwordStaffMinRunes = 8
	passwordMaxBytes      = 72
)

var ErrCredencialesInvalidas = errors.New("credenciales inválidas")

var dummyPasswordHash = func() string {
	hash, err := bcrypt.GenerateFromPassword([]byte("mesa-click-dummy-2026"), bcrypt.DefaultCost)
	if err != nil {
		panic("no se pudo inicializar la protección de credenciales")
	}
	return string(hash)
}()

// ValidarPassword aplica la política de la credencial local. El límite de 72
// bytes evita el truncamiento silencioso definido por bcrypt.
func ValidarPassword(password string) error {
	return validarPasswordConMinimo(password, passwordMinRunes)
}

// ValidarPasswordStaff aplica la política de claves operativas de equipo (mínimo 8 caracteres, letra + número).
func ValidarPasswordStaff(password string) error {
	return validarPasswordConMinimo(password, passwordStaffMinRunes)
}

func validarPasswordConMinimo(password string, minRunes int) error {
	if utf8.RuneCountInString(password) < minRunes {
		return fmt.Errorf("la contraseña debe tener al menos %d caracteres", minRunes)
	}
	if len([]byte(password)) > passwordMaxBytes {
		return fmt.Errorf("la contraseña no puede superar %d bytes", passwordMaxBytes)
	}

	tieneLetra := false
	tieneNumero := false
	for _, r := range password {
		tieneLetra = tieneLetra || unicode.IsLetter(r)
		tieneNumero = tieneNumero || unicode.IsDigit(r)
	}
	if !tieneLetra || !tieneNumero {
		return errors.New("la contraseña debe incluir al menos una letra y un número")
	}
	return nil
}

func HashPassword(password string) (string, error) {
	if err := ValidarPassword(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("no se pudo proteger la contraseña: %w", err)
	}
	return string(hash), nil
}

func HashPasswordStaff(password string) (string, error) {
	if err := ValidarPasswordStaff(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("no se pudo proteger la contraseña: %w", err)
	}
	return string(hash), nil
}

func CompararPassword(hash, password string) error {
	if hash == "" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return ErrCredencialesInvalidas
	}
	return nil
}
