package http

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/gofiber/contrib/v3/websocket"
	"golang.org/x/crypto/ssh"
)

type wsWriter struct {
	conn *websocket.Conn
}

func (w *wsWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if err := w.conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func sshClientConfig(user, password string) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境建议替换
		Timeout:         5 * time.Second,
	}
}

// ws://host/ws/ssh?host=192.168.1.10&port=22&user=root&token=xxx
func SSHWebsocket(c *websocket.Conn) {
	defer c.Close()

	// JWT 中间件已经跑过了，可以从 Locals 里拿用户信息
	userID := c.Locals("userId")
	username := c.Locals("username")
	_ = userID
	_ = username
	// TODO: 在这里做审计、权限控制等

	host := c.Query("host")
	if host == "" {
		_ = c.WriteMessage(websocket.TextMessage, []byte("missing host parameter\r\n"))
		return
	}
	port := c.Query("port")
	if port == "" {
		port = "22"
	}
	sshUser := c.Query("user")
	if sshUser == "" {
		sshUser = "root"
	}

	password := os.Getenv("SSH_DEFAULT_PASSWORD")
	if password == "" {
		_ = c.WriteMessage(websocket.TextMessage, []byte("SSH_DEFAULT_PASSWORD not set on server\r\n"))
		// 这里按需决定是否直接 return
	}

	addr := net.JoinHostPort(host, port)

	client, err := ssh.Dial("tcp", addr, sshClientConfig(sshUser, password))
	if err != nil {
		msg := fmt.Sprintf("failed to connect to %s: %v\r\n", addr, err)
		_ = c.WriteMessage(websocket.TextMessage, []byte(msg))
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte("failed to create ssh session\r\n"))
		return
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte("failed to get stdin pipe\r\n"))
		return
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte("failed to get stdout pipe\r\n"))
		return
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte("failed to get stderr pipe\r\n"))
		return
	}

	// 申请 PTY
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm", 25, 80, modes); err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte("failed to request pty\r\n"))
		return
	}

	if err := session.Shell(); err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte("failed to start shell\r\n"))
		return
	}

	// SSH → WebSocket
	go func() {
		writer := &wsWriter{conn: c}
		_, _ = io.Copy(writer, stdout)
	}()
	go func() {
		writer := &wsWriter{conn: c}
		_, _ = io.Copy(writer, stderr)
	}()

	// WebSocket → SSH
	for {
		msgType, data, err := c.ReadMessage()
		if err != nil {
			break
		}
		if msgType != websocket.TextMessage && msgType != websocket.BinaryMessage {
			continue
		}
		if len(data) == 0 {
			continue
		}
		if _, err := stdin.Write(data); err != nil {
			break
		}
	}

	_ = session.Close()
}
