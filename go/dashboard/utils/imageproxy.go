package utils

import (
	"strconv"
	"time"

	"github.com/TicketsBot-cloud/dashboard/config"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

func GenerateImageProxyToken(imageUrl string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"url":        imageUrl,
		"request_id": uuid.New().String(),
		"exp":        strconv.FormatInt(time.Now().Add(time.Second*30).Unix(), 10),
	})

	return token.SignedString([]byte(config.Conf.Bot.ImageProxySecret))
}
