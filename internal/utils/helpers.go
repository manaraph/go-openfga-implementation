package utils

import (
	"context"
	"strconv"

	"github.com/manaraph/go-openfga-implementation/pkg/middleware"
)

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(middleware.UserIdKey).(int)
	return strconv.Itoa(userID), ok
}
