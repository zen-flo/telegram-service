package telegram

import (
	"context"
	"errors"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"github.com/zen-flo/telegram-service/internal/session"
	"sync"
	"time"

	"github.com/gotd/td/telegram"
	"go.uber.org/zap"
)

type Client struct {
	appID   int
	appHash string
	logger  *zap.Logger

	mu     sync.RWMutex
	client *telegram.Client

	qrReqCh chan qrReq

	noop bool
}

type qrReq struct {
	resp       chan qrResp
	onAuthDone func()
}

type qrResp struct {
	url string
	err error
}

func NewClient(appID int, appHash string, logger *zap.Logger) *Client {
	c := &Client{
		appID:   appID,
		appHash: appHash,
		logger:  logger,
		qrReqCh: make(chan qrReq),
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
	c.client = telegram.NewClient(c.appID, c.appHash, telegram.Options{})
	c.mu.Unlock()

	return c.client.Run(ctx, func(runCtx context.Context) error {
		c.logger.Info("gotd run callback started")
		api := c.client.API()

		for {
			select {
			case <-runCtx.Done():
				c.logger.Info("gotd run callback exiting")
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
				}(req, qr)
			}
		}
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

	api := client.API()

	peerInput := &tg.InputPeerUser{
		UserID: 0,
	}

	_ = peer

	res, err := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
		Peer:     peerInput,
		Message:  text,
		RandomID: time.Now().UnixNano(),
	})

	if err != nil {
		return 0, err
	}

	switch v := res.(type) {
	case *tg.Updates:
		if len(v.Updates) > 0 {
			if m, ok := v.Updates[0].(*tg.UpdateMessageID); ok {
				return int64(m.ID), nil
			}
		}
	}

	return 0, nil
}

func (c *Client) DispatchMessage(
	session interface {
		PublishUpdate(*session.Message)
	},
	msg *session.Message,
) {
	session.PublishUpdate(msg)
}
