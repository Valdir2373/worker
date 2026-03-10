package socks5

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"golang.org/x/net/proxy"
)

const (
	socks5Ver       = 0x05
	socks5NoAuth     = 0x00
	socks5UserPass   = 0x02
	socks5NoAccept   = 0xFF
	socks5CmdConnect = 0x01
	socks5AtypIPv4   = 0x01
	socks5AtypDomain = 0x03
	socks5AtypIPv6   = 0x04
	socks5RepSuccess = 0x00
	socks5RepFailure = 0x01
	socks5RepCmdUnsupported  = 0x07
	socks5RepAtypUnsupported = 0x08
	socks5AuthVersion       = 0x01
	socks5AuthSuccess       = 0x00
	socks5AuthFailure       = 0x01
	socks5HandshakeTimeout  = 30 * time.Second
	socks5DialTimeout       = 60 * time.Second
)

type Server struct {
	addr   string
	dialer proxy.ContextDialer
	token  string // USER_PROXY_TOKEN; se não vazio, exige auth (password == token)
}

func NewServer(addr string, dialer proxy.ContextDialer) *Server {
	return &Server{addr: addr, dialer: dialer}
}

// NewServerWithToken cria servidor SOCKS5 que exige autenticação: cliente envia user/pass, password deve ser igual ao token.
func NewServerWithToken(addr string, dialer proxy.ContextDialer, token string) *Server {
	return &Server{addr: addr, dialer: dialer, token: token}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("socks5: %w", err)
	}
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	log.Printf("socks5: pronto em %s (auth=%v)", s.addr, s.token != "")
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				continue
			}
		}
		go s.handle(ctx, conn)
	}
}

func (s *Server) handle(ctx context.Context, clientConn net.Conn) {
	defer clientConn.Close()
	clientConn.SetDeadline(time.Now().Add(socks5HandshakeTimeout)) //nolint:errcheck
	if err := s.negotiateAuth(clientConn); err != nil {
		log.Printf("socks5: auth falhou: %v", err)
		return
	}
	target, err := s.readRequest(clientConn)
	if err != nil {
		return
	}
	clientConn.SetDeadline(time.Time{}) //nolint:errcheck
	dialCtx, dialCancel := context.WithTimeout(ctx, socks5DialTimeout)
	defer dialCancel()
	torConn, err := s.dialer.DialContext(dialCtx, "tcp", target)
	if err != nil {
		writeSocks5Reply(clientConn, socks5RepFailure) //nolint:errcheck
		return
	}
	defer torConn.Close()
	if err := writeSocks5Reply(clientConn, socks5RepSuccess); err != nil {
		return
	}
	tunnel(clientConn, torConn)
}

func (s *Server) negotiateAuth(conn net.Conn) error {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	if header[0] != socks5Ver {
		return fmt.Errorf("versão %d", header[0])
	}
	methods := make([]byte, int(header[1]))
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}

	if s.token != "" {
		hasUserPass := false
		for _, m := range methods {
			if m == socks5UserPass {
				hasUserPass = true
				break
			}
		}
		if !hasUserPass {
			conn.Write([]byte{socks5Ver, socks5NoAccept}) //nolint:errcheck
			return fmt.Errorf("token exigido: use método user/pass (password=USER_PROXY_TOKEN)")
		}
		conn.Write([]byte{socks5Ver, socks5UserPass}) //nolint:errcheck
		return s.authUserPass(conn)
	}

	for _, m := range methods {
		if m == socks5NoAuth {
			_, err := conn.Write([]byte{socks5Ver, socks5NoAuth})
			return err
		}
	}
	conn.Write([]byte{socks5Ver, socks5NoAccept}) //nolint:errcheck
	return fmt.Errorf("no auth")
}

func (s *Server) authUserPass(conn net.Conn) error {
	ver := make([]byte, 1)
	if _, err := io.ReadFull(conn, ver); err != nil {
		return err
	}
	if ver[0] != socks5AuthVersion {
		return fmt.Errorf("auth version %d", ver[0])
	}
	ulen := make([]byte, 1)
	if _, err := io.ReadFull(conn, ulen); err != nil {
		return err
	}
	user := make([]byte, int(ulen[0]))
	if _, err := io.ReadFull(conn, user); err != nil {
		return err
	}
	plen := make([]byte, 1)
	if _, err := io.ReadFull(conn, plen); err != nil {
		return err
	}
	pass := make([]byte, int(plen[0]))
	if _, err := io.ReadFull(conn, pass); err != nil {
		return err
	}
	if string(pass) != s.token {
		conn.Write([]byte{socks5AuthVersion, socks5AuthFailure}) //nolint:errcheck
		return fmt.Errorf("token inválido")
	}
	_, err := conn.Write([]byte{socks5AuthVersion, socks5AuthSuccess})
	return err
}

func (s *Server) readRequest(conn net.Conn) (string, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", err
	}
	if header[0] != socks5Ver || header[1] != socks5CmdConnect {
		writeSocks5Reply(conn, socks5RepCmdUnsupported) //nolint:errcheck
		return "", fmt.Errorf("comando não suportado")
	}
	var host string
	switch header[3] {
	case socks5AtypIPv4:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return "", err
		}
		host = net.IP(addr).String()
	case socks5AtypDomain:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return "", err
		}
		dom := make([]byte, int(lenBuf[0]))
		if _, err := io.ReadFull(conn, dom); err != nil {
			return "", err
		}
		host = string(dom)
	case socks5AtypIPv6:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return "", err
		}
		host = net.IP(addr).String()
	default:
		writeSocks5Reply(conn, socks5RepAtypUnsupported) //nolint:errcheck
		return "", fmt.Errorf("atyp não suportado")
	}
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return "", err
	}
	port := binary.BigEndian.Uint16(portBuf)
	return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
}

func writeSocks5Reply(conn net.Conn, rep byte) error {
	reply := []byte{socks5Ver, rep, 0x00, socks5AtypIPv4, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	_, err := conn.Write(reply)
	return err
}

func tunnel(clientConn, torConn net.Conn) {
	errc := make(chan error, 2)
	go func() { _, err := io.Copy(torConn, clientConn); errc <- err }()
	go func() { _, err := io.Copy(clientConn, torConn); errc <- err }()
	<-errc
	clientConn.Close()
	torConn.Close()
	<-errc
}

var _ IProxyServer = (*Server)(nil)
