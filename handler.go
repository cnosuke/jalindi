package main

import (
	"context"
	"time"

	"github.com/cnosuke/jalindi/pb"
	"github.com/golang/protobuf/ptypes"
	"github.com/lestrrat/go-fluent-client"
	"go.uber.org/zap"
)

type handler struct {
	jalindi.JalindiServiceServer
	fl     *fluent.Buffered
	tag    string
	logger *zap.SugaredLogger
}

func NewHandler(fl *fluent.Buffered, tag string, logger *zap.SugaredLogger) *handler {
	return &handler{
		fl:     fl,
		tag:    tag,
		logger: logger,
	}
}

func (h *handler) PostEvent(ctx context.Context, req *jalindi.PostEventRequest) (res *jalindi.PostEventResponse, err error) {
	h.logger.Infow("PostEvent", "request", req)

	err = h.post(
		req.Event,
		req.Client,
		req.BrowserUuid,
		req.RequestUuid,
		req.UserAgent,
		req.Referer,
	)

	if err != nil {
		h.logger.Errorw("PostEvent failed",
			"error", err,
			"event", req.Event,
			"client", req.Client,
			"browserUuid", req.BrowserUuid,
			"requestUuid", req.RequestUuid,
			"userAgent", req.UserAgent,
			"referer", req.Referer,
		)

		return nil, err
	} else {
		return &jalindi.PostEventResponse{}, nil
	}
}

func (h *handler) PostEventList(ctx context.Context, req *jalindi.PostEventListRequest) (res *jalindi.PostEventListResponse, err error) {
	h.logger.Infow("PostEventList", "request", req)

	for _, ev := range req.Events {
		err = h.post(
			ev,
			req.Client,
			req.BrowserUuid,
			req.RequestUuid,
			req.UserAgent,
			req.Referer,
		)

		if err != nil {
			h.logger.Errorw("PostEventList failed",
				"error", err,
				"event", ev,
				"client", req.Client,
				"browserUuid", req.BrowserUuid,
				"requestUuid", req.RequestUuid,
				"userAgent", req.UserAgent,
				"referer", req.Referer,
			)

			return nil, err
		}
	}

	return &jalindi.PostEventListResponse{}, nil
}

func (h *handler) post(event *jalindi.Event, client *jalindi.Client, browserUuid string, requestUuid string, userAgent string, referer string) (err error) {

	m := map[string]interface{}{
		"event": map[string]interface{}{
			"type":   event.Type,
			"group":  event.Group,
			"action": event.Action,
			"amount": event.Amount,
		},
		"client": map[string]interface{}{
			"name":             client.Name,
			"version":          client.Version,
			"platform":         client.Platform,
			"platform_version": client.PlatformVersion,
			"device_name":      client.DeviceName,
			"experiment":       client.Experiment,
		},
		"browser_uuid": browserUuid,
		"request_uuid": requestUuid,
		"user_agent":   userAgent,
		"referer":      referer,
	}

	var t time.Time
	if event.Timestamp != nil {
		t, err = ptypes.Timestamp(event.Timestamp)
		if err != nil {
			return err
		}
	} else {
		t = time.Now()
	}

	err = h.fl.Post(
		h.tag,
		m,
		fluent.WithTimestamp(t.In(time.UTC)),
		fluent.WithSyncAppend(true),
	)
	if err != nil {
		return err
	}

	return nil
}
