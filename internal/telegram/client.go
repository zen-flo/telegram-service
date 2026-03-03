package telegram

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tdtelegram "github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tgerr"

	"github.com/gotd/td/session"
	"github.com/gotd/td/tg"
	"github.com/zen-flo/telegram-service/internal/broker"
	"go.uber.org/zap"
)

type Client struct {
	appID   int
	appHash string
	logger  *zap.Logger

	mu     sync.RWMutex
	client *tdtelegram.Client

	qrReqCh chan qrReq
	noop    bool

	dispatcher *broker.Dispatcher
	sessionID  string

	peerMu       sync.RWMutex
	peerCache    map[string]tg.InputPeerClass
	loginTokenCh chan struct{}
}

type qrReq struct {
	resp       chan qrResp
	onAuthDone func()
}

type qrResp struct {
	url string
	err error
}

func NewClient(appID int, appHash string, logger *zap.Logger, dispatcher *broker.Dispatcher, sessionID string) *Client {
	c := &Client{
		appID:        appID,
		appHash:      appHash,
		logger:       logger,
		qrReqCh:      make(chan qrReq, 4),
		dispatcher:   dispatcher,
		sessionID:    sessionID,
		peerCache:    make(map[string]tg.InputPeerClass),
		loginTokenCh: make(chan struct{}, 1),
	}

	if appID == 0 || appHash == "" {
		c.noop = true
	}
	return c
}

func (c *Client) Start(ctx context.Context) error {
	if c.noop {
		c.logger.Debug("telegram client noop mode (no appID/appHash provided)")
		<-ctx.Done()
		return ctx.Err()
	}

	c.mu.Lock()
	c.client = tdtelegram.NewClient(
		c.appID,
		c.appHash,
		tdtelegram.Options{
			SessionStorage: &session.FileStorage{
				Path: "sessions/" + c.sessionID + ".json",
			},
			UpdateHandler: tdtelegram.UpdateHandlerFunc(func(ctx context.Context, u tg.UpdatesClass) error {
				c.handleUpdate(u)
				return nil
			}),
		},
	)
	c.mu.Unlock()

	return c.client.Run(ctx, func(runCtx context.Context) error {
		c.logger.Info("gotd run callback started")

		api := c.client.API()

		for {
			select {
			case <-runCtx.Done():
				c.logger.Info("gotd run callback exiting")

				if api != nil {
					if _, err := api.AuthLogOut(runCtx); err != nil {
						c.logger.Warn("auth.logOut failed during shutdown", zap.Error(err))
					} else {
						c.logger.Info("telegram client logged out")
					}
				}

				return runCtx.Err()

			case req := <-c.qrReqCh:

				go func(r qrReq) {

					pwd := os.Getenv("TG_2FA_PASSWORD")

					qr := qrlogin.NewQR(
						api,
						c.appID,
						c.appHash,
						qrlogin.Options{},
					)

					var urlSent bool

					_, err := qr.Auth(runCtx, qrlogin.LoggedIn(c.loginTokenCh),
						func(ctx context.Context, token qrlogin.Token) error {
							if !urlSent {
								r.resp <- qrResp{token.URL(), nil}
								urlSent = true
							} else {
								c.logger.Info("QR token refreshed",
									zap.String("url", token.URL()))
							}
							return nil
						},
					)

					if err != nil {
						if tgerr.Is(err, "SESSION_PASSWORD_NEEDED") {
							if pwd == "" {
								c.logger.Error("2FA password required but TG_2FA_PASSWORD is not set")
								return
							}
							if _, err := c.client.Auth().Password(runCtx, pwd); err != nil {
								c.logger.Error("2FA password auth failed", zap.Error(err))
								return
							}
							c.logger.Info("2FA password accepted")
						} else {
							if !urlSent {
								r.resp <- qrResp{"", err}
							}
							c.logger.Error("QR auth failed", zap.Error(err))
							return
						}
					}

					c.logger.Info("telegram auth success",
						zap.String("session", c.sessionID))

					if r.onAuthDone != nil {
						r.onAuthDone()
					}

					if c.dispatcher != nil {
						c.dispatcher.Publish(c.sessionID, &broker.Message{
							ID:        time.Now().UnixNano(),
							From:      "system",
							Text:      "authorized",
							Timestamp: time.Now().Unix(),
						})
					}

				}(req)
			}
		}
	})
}

func (c *Client) handleUpdate(update tg.UpdatesClass) {
	switch u := update.(type) {

	case *tg.Updates:
		for _, upd := range u.Updates {
			c.processSingleUpdate(upd)
		}

	case *tg.UpdateShort:
		c.processSingleUpdate(u.Update)

	case *tg.UpdateShortMessage:
		c.publishMessage(
			int64(u.ID),
			"unknown",
			u.Message,
			int64(u.Date),
		)

	case *tg.UpdateShortChatMessage:
		c.publishMessage(
			int64(u.ID),
			"chat",
			u.Message,
			int64(u.Date),
		)
	}
}

func (c *Client) processSingleUpdate(update tg.UpdateClass) {
	switch u := update.(type) {

	case *tg.UpdateLoginToken:
		_ = u
		// Сигналим каналу, что QR-код был отсканирован
		select {
		case c.loginTokenCh <- struct{}{}:
		default:
		}

	case *tg.UpdateNewMessage:
		msg, ok := u.Message.(*tg.Message)
		if !ok {
			return
		}

		if msg.Message == "" {
			return
		}

		from := "unknown"

		switch f := msg.FromID.(type) {
		case *tg.PeerUser:
			from = "user:" + strconv.FormatInt(f.UserID, 10)
		case *tg.PeerChat:
			from = "chat:" + strconv.FormatInt(f.ChatID, 10)
		case *tg.PeerChannel:
			from = "channel:" + strconv.FormatInt(f.ChannelID, 10)
		}

		c.publishMessage(
			int64(msg.ID),
			from,
			msg.Message,
			int64(msg.Date),
		)
	}
}

func (c *Client) publishMessage(id int64, from, text string, ts int64) {
	if c.dispatcher == nil {
		return
	}

	c.dispatcher.Publish(c.sessionID, &broker.Message{
		ID:        id,
		From:      from,
		Text:      text,
		Timestamp: ts,
	})
}

func (c *Client) StartQR(ctx context.Context, onAuthorized func()) (string, error) {
	if c.noop {
		return "", errors.New("telegram client is in noop mode")
	}

	req := qrReq{
		resp:       make(chan qrResp, 1),
		onAuthDone: onAuthorized,
	}

	select {
	case c.qrReqCh <- req:
	case <-ctx.Done():
		return "", ctx.Err()
	}

	select {
	case r := <-req.resp:
		return r.url, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *Client) SendMessage(
	ctx context.Context,
	peer string,
	text string,
) (int64, error) {

	if c.noop {
		return 0, errors.New("client is in noop mode")
	}

	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil {
		return 0, errors.New("client not started")
	}

	inputPeer, err := c.resolvePeer(ctx, peer)
	if err != nil {
		return 0, err
	}

	api := client.API()

	res, err := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
		Peer:     inputPeer,
		Message:  text,
		RandomID: time.Now().UnixNano(),
	})

	if err != nil {
		return 0, err
	}

	switch v := res.(type) {
	case *tg.Updates:
		for _, upd := range v.Updates {
			if m, ok := upd.(*tg.UpdateMessageID); ok {
				return int64(m.ID), nil
			}
		}
	}

	return 0, nil
}

func (c *Client) resolvePeer(ctx context.Context, peer string) (tg.InputPeerClass, error) {
	if peer == "" {
		return nil, errors.New("peer is empty")
	}

	normalized := strings.TrimPrefix(peer, "@")

	// check cache
	c.peerMu.RLock()
	if p, ok := c.peerCache[normalized]; ok {
		c.peerMu.RUnlock()
		return p, nil
	}
	c.peerMu.RUnlock()

	// resolve via the Telegram API
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil {
		return nil, errors.New("client not started")
	}

	api := client.API()

	res, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: normalized,
	})
	if err != nil {
		return nil, err
	}

	if len(res.Users) == 0 {
		return nil, errors.New("user not found")
	}

	u, ok := res.Users[0].(*tg.User)
	if !ok {
		return nil, errors.New("unexpected user type")
	}

	inputPeer := &tg.InputPeerUser{
		UserID:     u.ID,
		AccessHash: u.AccessHash,
	}

	// caching
	c.peerMu.Lock()
	c.peerCache[normalized] = inputPeer
	c.peerMu.Unlock()

	return inputPeer, nil
}
