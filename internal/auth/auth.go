package auth

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gotd/td/telegram"
	tgauth "github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"golang.org/x/term"
)

// ServiceConfig holds the dependencies for the auth service.
type ServiceConfig struct {
	SessionPath string
	APIID       int
	APIHash     string
}

// Service manages authentication lifecycle.
type Service struct {
	cfg     ServiceConfig
	session *FileSession
}

// NewService creates an auth Service.
func NewService(cfg ServiceConfig) *Service {
	return &Service{
		cfg:     cfg,
		session: NewFileSession(cfg.SessionPath),
	}
}

// LocalStatusResult is returned by LocalStatus.
type LocalStatusResult struct {
	Authenticated bool
	Session       SessionInfo
}

// SessionInfo holds the session file status.
type SessionInfo struct {
	Exists bool
	Path   string
}

// LocalStatus returns auth status derived from the local session file only (no network call).
func (s *Service) LocalStatus() LocalStatusResult {
	return LocalStatusResult{
		Authenticated: s.session.Exists(),
		Session: SessionInfo{
			Exists: s.session.Exists(),
			Path:   s.cfg.SessionPath,
		},
	}
}

// Logout removes the local session file.
func (s *Service) Logout() error {
	return s.session.Delete()
}

// Login runs an interactive MTProto login flow, prompting on stderr.
func (s *Service) Login(ctx context.Context) error {
	client := telegram.NewClient(s.cfg.APIID, s.cfg.APIHash, telegram.Options{
		SessionStorage: s.session,
	})

	return client.Run(ctx, func(ctx context.Context) error {
		status, err := client.Auth().Status(ctx)
		if err != nil {
			return fmt.Errorf("checking auth status: %w", err)
		}
		if status.Authorized {
			fmt.Fprintln(os.Stderr, "Already authenticated.")
			return nil
		}

		flow := tgauth.NewFlow(
			&consoleAuthenticator{},
			tgauth.SendCodeOptions{},
		)
		return client.Auth().IfNecessary(ctx, flow)
	})
}

// GetAccountInfo fetches the authenticated account's username and phone (redacted).
func (s *Service) GetAccountInfo(ctx context.Context) (username, phoneRedacted string, err error) {
	client := telegram.NewClient(s.cfg.APIID, s.cfg.APIHash, telegram.Options{
		SessionStorage: s.session,
	})
	err = client.Run(ctx, func(ctx context.Context) error {
		self, err := client.Self(ctx)
		if err != nil {
			return err
		}
		username = self.Username
		phoneRedacted = redactPhone(self.Phone)
		return nil
	})
	return
}

// consoleAuthenticator prompts the operator on stderr.
type consoleAuthenticator struct {
	phone string
}

func (c *consoleAuthenticator) Phone(_ context.Context) (string, error) {
	if c.phone != "" {
		return c.phone, nil
	}
	fmt.Fprint(os.Stderr, "Phone number (+CCNUMBER): ")
	var phone string
	fmt.Fscan(os.Stdin, &phone)
	c.phone = strings.TrimSpace(phone)
	return c.phone, nil
}

func (c *consoleAuthenticator) Password(_ context.Context) (string, error) {
	fmt.Fprint(os.Stderr, "2FA password: ")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

func (c *consoleAuthenticator) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	fmt.Fprint(os.Stderr, "Telegram code: ")
	var code string
	fmt.Fscan(os.Stdin, &code)
	return strings.TrimSpace(code), nil
}

func (c *consoleAuthenticator) AcceptTermsOfService(_ context.Context, _ tg.HelpTermsOfService) error {
	return nil
}

func (c *consoleAuthenticator) SignUp(_ context.Context) (tgauth.UserInfo, error) {
	return tgauth.UserInfo{}, fmt.Errorf("sign-up not supported by tg-alerts")
}

// redactPhone masks digits except the last 4, e.g. "+55******1234".
func redactPhone(phone string) string {
	runes := []rune(phone)
	digitCount := 0
	for _, r := range runes {
		if r >= '0' && r <= '9' {
			digitCount++
		}
	}
	keep := 4
	masked := 0
	out := make([]rune, 0, len(runes))
	for _, r := range runes {
		if r >= '0' && r <= '9' {
			if digitCount-masked > keep {
				out = append(out, '*')
				masked++
			} else {
				out = append(out, r)
			}
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}
