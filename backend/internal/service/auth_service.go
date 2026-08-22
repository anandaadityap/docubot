package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/supernand/docubot/backend/internal/models"
	"github.com/supernand/docubot/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

const (
	jwtIssuer      = "docubot"
	jwtTTL         = 24 * time.Hour
	minPasswordLen = 8
)

var (
	// ErrValidation is returned for invalid register/login input.
	ErrValidation = errors.New("validation error")
	// ErrInvalidCredentials is returned for bad email/password (or unknown email).
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUnauthorized is returned when a token is missing/invalid/expired.
	ErrUnauthorized = errors.New("unauthorized")
)

// AuthService handles registration, login, and JWT helpers.
type AuthService struct {
	users  repository.UserRepository
	secret []byte
	now    func() time.Time
}

// NewAuthService constructs an AuthService.
func NewAuthService(users repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		users:  users,
		secret: []byte(jwtSecret),
		now:    time.Now,
	}
}

// Register creates a new admin user (password hashed with bcrypt).
func (s *AuthService) Register(ctx context.Context, name, email, password string) (*models.User, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	if len(password) < minPasswordLen {
		return nil, fmt.Errorf("%w: password must be at least %d characters", ErrValidation, minPasswordLen)
	}
	name = strings.TrimSpace(name)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.Create(ctx, email, string(hash), name)
	if err != nil {
		if errors.Is(err, repository.ErrEmailTaken) {
			return nil, err
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

// LoginResult is returned on successful login.
type LoginResult struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// Login verifies credentials and returns a JWT + user profile.
func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	if password == "" {
		return nil, fmt.Errorf("%w: password is required", ErrValidation)
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.issueToken(user)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &LoginResult{Token: token, User: user}, nil
}

// Me loads the user profile by ID (from JWT claims).
func (s *AuthService) Me(ctx context.Context, userID int64) (*models.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

// Claims are JWT claims for DocuBot auth.
type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// ParseToken validates a JWT and returns the user ID from sub.
func (s *AuthService) ParseToken(tokenStr string) (int64, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return 0, ErrUnauthorized
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return 0, ErrUnauthorized
	}

	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrUnauthorized
	}
	return id, nil
}

func (s *AuthService) issueToken(user *models.User) (string, error) {
	now := s.now()
	claims := Claims{
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(user.ID, 10),
			Issuer:    jwtIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(jwtTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func normalizeEmail(email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return "", errors.New("email is required")
	}
	if strings.ContainsAny(email, " <>") {
		return "", errors.New("invalid email format")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return "", errors.New("invalid email format")
	}
	return strings.ToLower(addr.Address), nil
}
