package cliauth

import (
	"context"
	"net/http"

	"github.com/common-fate/clio"
	"golang.org/x/oauth2"
)

type Response struct {
	// Err is set if there was an error which
	// prevented the flow from completing
	Err          error
	Token        *oauth2.Token
	DashboardURL string
}

type Server struct {
	Response chan Response
}

func NewServer() *Server {
	return &Server{
		Response: nil,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/oauth/login", s.oauthLogin)
	mux.HandleFunc("/oauth/login/callback", s.oauthCallback)

	return mux
}

func (s *Server) oauthLogin(w http.ResponseWriter, r *http.Request) {
	conf := NewAuthConfig()

	u := conf.AuthCodeURL(generateStateOauthCookie(w))
	// Add Cache-Control header to prevent caching of the response
	w.Header().Set("Cache-Control", "no-cache")

	http.Redirect(w, r, u, http.StatusPermanentRedirect)
}

func (s *Server) oauthCallback(w http.ResponseWriter, r *http.Request) {
	// Read oauthState from Cookie
	oauthState, _ := r.Cookie("oauthstate")

	queryParams := r.URL.Query()

	state := queryParams.Get("state")
	if state == "" || oauthState == nil || state != oauthState.Value {
		clio.Error("invalid oauth2 state")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)

		return
	}

	code := queryParams.Get("code")
	if code == "" {
		clio.Error("oauth2 code empty")
		http.Redirect(w, r, "/", http.StatusBadRequest)
		return
	}

	conf := NewAuthConfig()

	// Exchange will do the handshake to retrieve the initial access token.
	tok, err := conf.Exchange(context.Background(), code)
	if err != nil {
		s.Response <- Response{Err: err}
		_, err = w.Write([]byte("there was a problem logging in to Common Fate Provider Registry: " + err.Error()))
		http.Redirect(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: overwriting access token as IDToken as cognito authorizer
	// expects the Authorization Header to contain IDToken.
	IDToken, ok := tok.Extra("id_token").(string)
	if !ok {
		return
	}

	tok.AccessToken = IDToken
	clio.Debugw("token", "token", IDToken)
	_, err = w.Write([]byte("logged in to Common Fate Provider Registry successfully! You can close this window."))
	if err != nil {
		clio.Errorf("write error: %s", err.Error())
	}

	response := Response{
		Token: tok,
	}

	s.Response <- response
}
