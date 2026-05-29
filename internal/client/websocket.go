package client

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/cbeuw/Cloak/internal/common"
	"github.com/gorilla/websocket"
	utls "github.com/refraction-networking/utls"
)

type WSOverTLS struct {
	*common.WebSocketConn
	wsUrl string
}

func (ws *WSOverTLS) Handshake(rawConn net.Conn, authInfo AuthInfo) (sessionKey [32]byte, err error) {
	var caCertPool *x509.CertPool
	caCertPool = nil
	if authInfo.CAPath != "" && !authInfo.InsecureSkipVerify {
		// 1. 加载你的自定义 CA 证书
		caCert, err := os.ReadFile(authInfo.CAPath)
		if err != nil {
			return sessionKey, fmt.Errorf("failed to read CA file: %v", err)
		}

		// 2. 创建证书池并添加 CA
		caCertPool = x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return sessionKey, errors.New("failed to parse CA certificate")
		}
	}
	utlsConfig := &utls.Config{
		ServerName:         authInfo.MockDomain,
		InsecureSkipVerify: authInfo.InsecureSkipVerify,
		RootCAs:            caCertPool,
	}

	u, err := url.Parse(ws.wsUrl)
	if err != nil {
		return sessionKey, fmt.Errorf("failed to parse ws url: %v", err)
	}

	payload, sharedSecret := makeAuthenticationPayload(authInfo)
	header := http.Header{}
	header.Add("hidden", base64.StdEncoding.EncodeToString(append(payload.randPubKey[:], payload.ciphertextWithTag[:]...)))
	var dialer = websocket.Dialer{
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			uconn := utls.UClient(rawConn, utlsConfig, utls.HelloChrome_Auto)
			err = uconn.BuildHandshakeState()
			if err != nil {
				return nil, err
			}
			for i, extension := range uconn.Extensions {
				_, ok := extension.(*utls.ALPNExtension)
				if ok {
					uconn.Extensions = append(uconn.Extensions[:i], uconn.Extensions[i+1:]...)
					break
				}
			}
			err = uconn.Handshake()
			return uconn, err
		},
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   32768,
		WriteBufferSize:  32768,
		WriteBufferPool:  &sync.Pool{},
	}
	c, _, err := dialer.Dial(u.String(), header)
	if err != nil {
		return sessionKey, fmt.Errorf("failed to handshake: %v", err)
	}

	ws.WebSocketConn = &common.WebSocketConn{Conn: c}

	buf := make([]byte, 128)
	n, err := ws.Read(buf)
	if err != nil {
		return sessionKey, fmt.Errorf("failed to read reply: %v", err)
	}

	if n != 60 {
		return sessionKey, errors.New("reply must be 60 bytes")
	}

	reply := buf[:60]
	sessionKeySlice, err := common.AESGCMDecrypt(reply[:12], sharedSecret[:], reply[12:])
	if err != nil {
		return
	}
	copy(sessionKey[:], sessionKeySlice)

	return
}

func (ws *WSOverTLS) Close() error {
	if ws.WebSocketConn != nil {
		return ws.WebSocketConn.Close()
	}
	return nil
}
