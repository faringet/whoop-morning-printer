package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/services/morningbot/config"
	"github.com/gin-gonic/gin"
)

var (
	errAuthorizationRequired = errors.New("telegram authorization is required")
	errAuthorizationInvalid  = errors.New("telegram authorization is invalid")
	errAuthorizationExpired  = errors.New("telegram authorization has expired")
	errUserForbidden         = errors.New("telegram user is not allowed")
)

const authorizationScheme = "tma"

type AuthMiddleware struct {
	botToken       string
	allowedUserIDs map[int64]struct{}
	authMaxAge     time.Duration
	devAuth        config.DevAuth
	now            func() time.Time
}

type telegramUser struct {
	ID int64 `json:"id"`
}

func NewAuthMiddleware(botToken string, allowedUserIDs []int64, miniAppConfig config.MiniApp) (*AuthMiddleware, error) {
	botToken = strings.TrimSpace(botToken)
	if botToken == "" {
		return nil, errors.New("morningbot httpapi auth: bot token is required")
	}
	if miniAppConfig.AuthMaxAge <= 0 {
		return nil, errors.New("morningbot httpapi auth: auth max age must be > 0")
	}

	allowed := make(map[int64]struct{}, len(allowedUserIDs))
	for _, userID := range allowedUserIDs {
		if userID <= 0 {
			return nil, fmt.Errorf("morningbot httpapi auth: invalid allowed user id: %d", userID)
		}

		allowed[userID] = struct{}{}
	}

	if len(allowed) == 0 {
		return nil, errors.New("morningbot httpapi auth: allowed user ids must not be empty")
	}

	if miniAppConfig.DevAuth.Enabled {
		if miniAppConfig.DevAuth.TelegramUserID <= 0 {
			return nil, errors.New("morningbot httpapi auth: dev telegram user id must be > 0")
		}
		if _, ok := allowed[miniAppConfig.DevAuth.TelegramUserID]; !ok {
			return nil, errors.New("morningbot httpapi auth: dev telegram user id is not allowed")
		}
	}

	return &AuthMiddleware{
		botToken:       botToken,
		allowedUserIDs: allowed,
		authMaxAge:     miniAppConfig.AuthMaxAge,
		devAuth:        miniAppConfig.DevAuth,
		now:            time.Now,
	}, nil
}

func (a *AuthMiddleware) RequireTelegramUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		if a == nil {
			writeError(c, 500, "internal_error", "Authorization middleware is unavailable")
			return
		}

		telegramUserID, err := a.authenticate(c.GetHeader("Authorization"))
		if err != nil {
			switch {
			case errors.Is(err, errAuthorizationRequired):
				writeError(c, 401, "telegram_auth_required", "Telegram authorization is required")

			case errors.Is(err, errAuthorizationExpired):
				writeError(c, 401, "telegram_auth_expired", "Telegram authorization has expired")

			case errors.Is(err, errUserForbidden):
				writeError(c, 403, "telegram_user_forbidden", "Telegram user is not allowed")

			default:
				writeError(c, 401, "telegram_auth_invalid", "Telegram authorization is invalid")
			}

			return
		}

		setTelegramUserID(c, telegramUserID)
		c.Next()
	}
}

func (a *AuthMiddleware) authenticate(authorizationHeader string) (int64, error) {
	authorizationHeader = strings.TrimSpace(authorizationHeader)

	if authorizationHeader == "" {
		if a.devAuth.Enabled {
			return a.authorizeUser(a.devAuth.TelegramUserID)
		}

		return 0, errAuthorizationRequired
	}

	initData, err := parseAuthorizationHeader(authorizationHeader)
	if err != nil {
		return 0, err
	}

	telegramUserID, err := a.validateInitData(initData)
	if err != nil {
		return 0, err
	}

	return a.authorizeUser(telegramUserID)
}

func (a *AuthMiddleware) validateInitData(initData string) (int64, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return 0, fmt.Errorf("%w: parse init data: %v", errAuthorizationInvalid, err)
	}

	hashValue, err := singleQueryValue(values, "hash")
	if err != nil {
		return 0, err
	}

	receivedHash, err := hex.DecodeString(hashValue)
	if err != nil || len(receivedHash) != sha256.Size {
		return 0, errAuthorizationInvalid
	}

	dataCheckString, err := buildDataCheckString(values)
	if err != nil {
		return 0, err
	}

	secretKeyHMAC := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secretKeyHMAC.Write([]byte(a.botToken))
	secretKey := secretKeyHMAC.Sum(nil)

	expectedHashHMAC := hmac.New(sha256.New, secretKey)
	_, _ = expectedHashHMAC.Write([]byte(dataCheckString))
	expectedHash := expectedHashHMAC.Sum(nil)

	if !hmac.Equal(receivedHash, expectedHash) {
		return 0, errAuthorizationInvalid
	}

	authDateValue, err := singleQueryValue(values, "auth_date")
	if err != nil {
		return 0, err
	}

	authDateUnix, err := strconv.ParseInt(authDateValue, 10, 64)
	if err != nil || authDateUnix <= 0 {
		return 0, errAuthorizationInvalid
	}

	authDate := time.Unix(authDateUnix, 0).UTC()
	now := a.now().UTC()

	if authDate.After(now.Add(time.Minute)) {
		return 0, errAuthorizationInvalid
	}

	if now.Sub(authDate) > a.authMaxAge {
		return 0, errAuthorizationExpired
	}

	userValue, err := singleQueryValue(values, "user")
	if err != nil {
		return 0, err
	}

	var user telegramUser
	if err := json.Unmarshal([]byte(userValue), &user); err != nil {
		return 0, fmt.Errorf("%w: decode user: %v", errAuthorizationInvalid, err)
	}
	if user.ID <= 0 {
		return 0, errAuthorizationInvalid
	}

	return user.ID, nil
}

func (a *AuthMiddleware) authorizeUser(telegramUserID int64) (int64, error) {
	if telegramUserID <= 0 {
		return 0, errAuthorizationInvalid
	}

	if _, ok := a.allowedUserIDs[telegramUserID]; !ok {
		return 0, errUserForbidden
	}

	return telegramUserID, nil
}

func parseAuthorizationHeader(value string) (string, error) {
	separatorIndex := strings.IndexByte(value, ' ')
	if separatorIndex <= 0 {
		return "", errAuthorizationInvalid
	}

	scheme := strings.TrimSpace(value[:separatorIndex])
	initData := strings.TrimSpace(value[separatorIndex+1:])

	if !strings.EqualFold(scheme, authorizationScheme) || initData == "" {
		return "", errAuthorizationInvalid
	}

	return initData, nil
}

func buildDataCheckString(values url.Values) (string, error) {
	keys := make([]string, 0, len(values))

	for key := range values {
		if key == "hash" {
			continue
		}

		keys = append(keys, key)
	}

	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		value, err := singleQueryValue(values, key)
		if err != nil {
			return "", err
		}

		pairs = append(pairs, key+"="+value)
	}

	return strings.Join(pairs, "\n"), nil
}

func singleQueryValue(values url.Values, key string) (string, error) {
	items, ok := values[key]
	if !ok || len(items) != 1 || strings.TrimSpace(items[0]) == "" {
		return "", errAuthorizationInvalid
	}

	return items[0], nil
}
