// Copyright (c) nomadsre
// SPDX-License-Identifier: MPL-2.0

// Package client wraps *discordgo.Session so resource implementations do not
// import discordgo directly. The wrapper is intentionally thin: it only holds
// the session and exposes small helpers for behavior we need in more than one
// place (currently REST-only; the gateway is never opened).
package client

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

// Client wraps a configured *discordgo.Session.
type Client struct {
	Session *discordgo.Session
}

// New constructs a client authenticated with the given bot token. The returned
// session is REST-only; callers MUST NOT invoke Session.Open().
func New(token, userAgent string) (*Client, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("construct discord session: %w", err)
	}
	if userAgent != "" {
		s.UserAgent = userAgent
	}
	return &Client{Session: s}, nil
}

// IsNotFound reports whether err is a Discord REST 404 response. Resources use
// this in Read to detect drift (object deleted out-of-band) and remove the
// resource from state instead of surfacing an error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var rerr *discordgo.RESTError
	if errors.As(err, &rerr) && rerr.Response != nil {
		return rerr.Response.StatusCode == http.StatusNotFound
	}
	return false
}
