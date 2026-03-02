package telegram

import (
	"context"
	"errors"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"github.com/zen-flo/telegram-service/internal/broker"
	"strconv"
	"strings"
	"sync"
	"time"

	tdtelegram "github.com/gotd/td/telegram"
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

	peerMu    sync.RWMutex
	peerCache map[string]tg.InputPeerClass
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
		appID:      appID,
		appHash:    appHash,
		logger:     logger,
		qrReqCh:    make(chan qrReq, 4),
		dispatcher: dispatcher,
		sessionID:  sessionID,
		peerCache:  make(map[string]tg.InputPeerClass),
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

				// attempt logout before shutdown
				_, err := api.AuthLogOut(runCtx)
				if err != nil {
					c.logger.Warn("auth.logOut failed during shutdown", zap.Error(err))
				} else {
					c.logger.Info("telegram client logged out")
				}

				return runCtx.Err()

			case req := <-c.qrReqCh:
				qr := qrlogin.NewQR(api, c.appID, c.appHash, qrlogin.Options{})

				token, err := qr.Export(runCtx)
				if err != nil {
					req.resp <- qrResp{"", err}
					continue
				}

				url := token.URL()
				req.resp <- qrResp{url, nil}

				go func(r qrReq, q qrlogin.QR) {
					_, err := q.Import(runCtx)
					if err != nil {
						c.logger.Error("qr import failed", zap.Error(err))
						return
					}
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
				}(req, qr)
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
