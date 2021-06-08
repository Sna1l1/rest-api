package util

import (
	"context"
	"fmt"
	"regexp"
	"unicode"

	"github.com/alexedwards/argon2id"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func HashPsw(pass, repass string) (string, error) {
	if pass != repass {
		return "", fmt.Errorf("passwords don`t match")
	}

	var (
		upp, low, num bool
		tot           uint8
	)

	for _, char := range pass {
		switch {
		case unicode.IsUpper(char):
			upp = true
			tot++
		case unicode.IsLower(char):
			low = true
			tot++
		case unicode.IsNumber(char):
			num = true
			tot++
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			tot++
		default:
			return "", fmt.Errorf("unknown symbols")
		}
	}

	if !upp || !low || !num || tot < 8 || tot > 32 {
		return "", fmt.Errorf("incorrect lenght or scheme of password")
	}

	hash, err := argon2id.CreateHash(pass, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hash, nil
}
func ValidateUsername(client *mongo.Client, ctx context.Context, username string) error {
	var (
		upp, low  bool
		tot, spec uint8
	)

	for _, char := range username {
		switch {
		case unicode.IsUpper(char):
			upp = true
			tot++
		case unicode.IsLower(char):
			low = true
			tot++
		case unicode.IsNumber(char):
			tot++
		case char == '!' || char == '@':
			tot++
			spec++
		default:
			return fmt.Errorf("unknown symbols")
		}
	}
	if tot < 3 {
		return fmt.Errorf("login length must be more than 3 characters")
	}
	if tot > 16 {
		return fmt.Errorf("login lenght cannot be more that 16 characters")
	}
	if !upp && !low {
		return fmt.Errorf("login must contain at least 1 letter")
	}
	if spec > 1 {
		return fmt.Errorf("login cannot contain more than 1 special symbol")
	}
	collection := client.Database("sample_analytics").Collection("test")

	if err := collection.FindOne(ctx, bson.M{"username": username}); err == nil {
		return fmt.Errorf("username already taken")
	}
	return nil
}
func ValidateEmail(client *mongo.Client, ctx context.Context, email string) error {
	pattern := "(?:[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*|\"(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21\x23-\x5b\x5d-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])*\")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\\[(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?|[a-z0-9-]*[a-z0-9]:(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21-\x5a\x53-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])+)\\])"

	matched, err := regexp.Match(pattern, []byte(email))
	if err != nil {
		return err
	}

	if !matched {
		return fmt.Errorf("invalid email")
	}

	collection := client.Database("sample_analytics").Collection("test")

	if err := collection.FindOne(ctx, bson.M{"email": email}); err == nil {
		return fmt.Errorf("email already in use")
	}

	return nil
}
