package chat

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
)

// loginReq represents login request from client.
type loginReq struct {
	Type     float64 `json:"type"`
	Nickname string  `json:"nickname"`
}

// loginResp represents login response to client.
type loginResp struct {
	Type   float64 `json:"type"`
	Token  string  `json:"token"`
	Status float64 `json:"status"`
}

// postMsgReq respresents post message request from client.
type postMsgReq struct {
	Type  float64 `json:"type"`
	Token string  `json:"token"`
	Msg   string  `json:"msg"`
}

// postMsgResp represents post message response to client.
type postMsgResp struct {
	Type   float64 `json:"type"`
	Status float64 `json:"status"`
}

// chatMsgToClient represents message to print in client's chat box.
type chatMsgToClient struct {
	Type     float64 `json:"type"`
	Nickname string  `json:"nickname"`
	Msg      string  `json:"msg"`
	IsSystem bool    `json:"isSystem"`
}

// onlineUsersReq represents request for list of online users from client.
type onlineUsersReq struct {
	Type  float64 `json:"type"`
	Token string  `json:"token"`
}

// onlineUsers represent list of online users to send to client.
type onlineUsers struct {
	Type   float64  `json:"type"`
	Status float64  `json:"status"`
	Users  []string `json:"users"`
}

// used to distinguish between types of various JSON requests and responses.
const (
	typeLoginReq float64 = iota + 1
	typeLoginResp
	typePostMessageReq
	typePostMessageResp
	typeChatMessageToClient
	typeOnlineUsersReq
	typeOnlineUsers
)

// represents various statuses to send in responses to client.
const (
	statusOk float64 = iota + 1
	statusInvalidToken
	statusNameAlreadyTaken
	statusNameIsEmpty
	statusNameIsTooLong
	statusMessageIsEmpty
	statusMessageIsTooLong
)

// client represents data stored per logged in client.
type client struct {
	nickname string
	conn     *websocket.Conn
}

// Handler represents chat handler.
type Handler struct {
	log     *logrus.Logger
	clients sync.Map   // map of access token to client struct
	mut     sync.Mutex // used to protect from concurrent websocket write
}

// NewHandler returns new chat handler
func NewHandler(log *logrus.Logger) Handler {
	return Handler{log: log, clients: sync.Map{}}
}

// Chat is a main handler function for server.
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error(errors.Wrap(err, "Websocket upgrade connection"))
		return
	}
	defer conn.Close()

	token := uuid.NewString()
	loggedIn := false
	h.log.WithFields(logrus.Fields{"token": token, "addr": conn.RemoteAddr()}).Info("Received connection")

	for {
		var req map[string]any
		err := conn.ReadJSON(&req)
		var closeErr *websocket.CloseError
		var netErr net.Error

		name := h.nickname(token)

		if errors.As(err, &closeErr) || errors.As(err, &netErr) {
			h.handleDisconnect(token, name, loggedIn, conn)
			break
		} else if err != nil {
			h.log.Error(errors.Wrap(err, "Read JSON from connection"))
		}

		switch req["type"] {
		case typePostMessageReq:
			h.handlePostMessageReq(req, token, name, conn)
		case typeLoginReq:
			loggedIn = h.handleLoginReq(req, token, loggedIn, conn)
		case typeOnlineUsersReq:
			h.handleOnlineUsersReq(req, token, name, conn)
		default:
			h.log.WithFields(logrus.Fields{
				"token": token,
				"name":  name,
				"addr":  conn.RemoteAddr(),
				"req":   req,
			}).Warn("Received unknown request")
		}
	}
}

// handleDisconnect performs actions to do when client disconnects from server.
func (h *Handler) handleDisconnect(token string, name string, loggedIn bool, conn *websocket.Conn) {
	h.log.WithFields(logrus.Fields{
		"token": token,
		"name":  name,
		"addr":  conn.RemoteAddr(),
	}).Info("Client disconnected")
	if loggedIn {
		h.clients.Delete(token)
		h.broadcast(chatMsgToClient{
			Type:     typeChatMessageToClient,
			Msg:      fmt.Sprintf("%v left the chat", name),
			IsSystem: true,
		})
		h.broadcast(onlineUsers{Type: typeOnlineUsers, Status: statusOk, Users: h.loggedInUsers()})
	}
}

// handlePostMessageReq performs actions to do when client requests to post message.
func (h *Handler) handlePostMessageReq(r map[string]any, token string, name string, conn *websocket.Conn) {
	var req postMsgReq
	err := mapstructure.Decode(r, &req)
	if err != nil {
		h.log.Error(errors.Wrap(err, "Decode post message request"))
		return
	}
	if req.Msg == "" {
		return
	}

	h.log.WithFields(logrus.Fields{
		"token": token,
		"name":  name,
		"addr":  conn.RemoteAddr(),
		"req":   req,
	}).Debug("Received post message request")

	resp := postMsgResp{Type: typePostMessageResp, Status: statusOk}
	_, hasToken := h.clients.Load(token)
	if !hasToken {
		resp.Status = statusInvalidToken
	} else if len(req.Msg) == 0 {
		resp.Status = statusMessageIsEmpty
	} else if len(req.Msg) > 2000 {
		resp.Status = statusMessageIsTooLong
	}
	err = h.send(conn, resp)
	if err != nil {
		h.log.Error(errors.Wrap(err, "Write post message status response"))
		return
	}
	if resp.Status != statusOk {
		return
	}

	h.broadcast(chatMsgToClient{Type: typeChatMessageToClient, Nickname: name, Msg: req.Msg})
}

// handleLoginReq performs actions to do when client tries to login. It returns true if client already logged in or
// just did it successfully.
func (h *Handler) handleLoginReq(r map[string]any, token string, loggedIn bool, conn *websocket.Conn) bool {
	var req loginReq
	err := mapstructure.Decode(r, &req)
	if err != nil {
		h.log.Error(errors.Wrap(err, "Decode login request"))
		return loggedIn
	}
	h.log.WithFields(logrus.Fields{"token": token, "addr": conn.RemoteAddr()}).Debug("Received login request")

	resp := loginResp{Type: typeLoginResp}
	if h.hasNickname(req.Nickname) {
		resp.Status = statusNameAlreadyTaken
	} else if len(req.Nickname) == 0 {
		resp.Status = statusNameIsEmpty
	} else if len(req.Nickname) > 20 {
		resp.Status = statusNameIsTooLong
	} else {
		resp.Status = statusOk
		resp.Token = token

		h.clients.Store(token, client{nickname: req.Nickname, conn: conn})
		loggedIn = true

		h.log.WithFields(logrus.Fields{
			"token": token,
			"name":  req.Nickname,
			"addr":  conn.RemoteAddr(),
		}).Info("Client logged in")

		h.broadcast(chatMsgToClient{
			Type:     typeChatMessageToClient,
			Msg:      fmt.Sprintf("%v joined the chat", req.Nickname),
			IsSystem: true,
		})
		h.broadcast(onlineUsers{Type: typeOnlineUsers, Status: statusOk, Users: h.loggedInUsers()})
	}
	err = h.send(conn, resp)
	if err != nil {
		h.log.Error(errors.Wrap(err, "Write login status response"))
	}

	return loggedIn
}

// handleOnlineUsersReq preforms actions to do when client requests list of online users.
func (h *Handler) handleOnlineUsersReq(r map[string]any, token string, name string, conn *websocket.Conn) {
	var req onlineUsersReq
	err := mapstructure.Decode(r, &req)
	if err != nil {
		h.log.Error(errors.Wrap(err, "Decode online users request"))
		return
	}
	h.log.WithFields(logrus.Fields{
		"token": token,
		"name":  name,
		"addr":  conn.RemoteAddr(),
		"req":   req,
	}).Debug("Received online users request")

	_, hasToken := h.clients.Load(token)
	resp := onlineUsers{Type: typeOnlineUsers, Status: lo.Ternary(hasToken, statusOk, statusInvalidToken)}
	if hasToken {
		resp.Users = h.loggedInUsers()
	}
	err = h.send(conn, resp)
	if err != nil {
		h.log.Error(errors.Wrap(err, "Write online users response"))
	}
}

// hasNickname returns true if <nick> is found among logged in clients.
func (h *Handler) hasNickname(nick string) bool {
	var has bool
	h.clients.Range(func(_, val any) bool {
		if val.(client).nickname == nick {
			has = true
			return false
		}
		return true
	})
	return has
}

// loggedInUsers returns list of logged in users.
func (h *Handler) loggedInUsers() []string {
	out := []string{}
	h.clients.Range(func(_, val any) bool {
		out = append(out, val.(client).nickname)
		return true
	})
	return out
}

// broadcast sends <msg> to every logged in client.
func (h *Handler) broadcast(msg any) {
	h.clients.Range(func(_, val any) bool {
		if err := h.send(val.(client).conn, msg); err != nil {
			h.log.Error(errors.Wrap(err, "Broadcast message"))
		}
		return true
	})
}

// nickname returns nickname for user with access <token> or empty string if not found.
func (h *Handler) nickname(token string) string {
	if c, ok := h.clients.Load(token); ok {
		return c.(client).nickname
	}
	return ""
}

// send sends JSON <msg> to <conn>, guarding this operation with mutex lock / unlock.
func (h *Handler) send(conn *websocket.Conn, msg any) error {
	h.mut.Lock()
	defer h.mut.Unlock()
	return conn.WriteJSON(msg)
}
