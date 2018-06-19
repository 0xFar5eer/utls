// Copyright 2017 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tls

import (
	"io"
)

type TLSExtension interface {
	writeToUConn(*UConn) error

	Len() int // includes header

	// Read reads up to len(p) bytes into p.
	// It returns the number of bytes read (0 <= n <= len(p)) and any error encountered.
	Read(p []byte) (n int, err error) // implements io.Reader
}

type NPNExtension struct {
	NextProtos []string
}

func (e *NPNExtension) writeToUConn(uc *UConn) error {
	uc.config.NextProtos = e.NextProtos
	uc.HandshakeState.Hello.NextProtoNeg = true
	return nil
}

func (e *NPNExtension) Len() int {
	return 4
}

func (e *NPNExtension) Read(b []byte) (int, error) {
	if len(b) < e.Len() {
		return 0, io.ErrShortBuffer
	}
	b[0] = byte(extensionNextProtoNeg >> 8)
	b[1] = byte(extensionNextProtoNeg & 0xff)
	// The length is always 0
	return e.Len(), io.EOF
}

type SNIExtension struct {
	ServerName string // not an array because go crypto/tls doesn't support multiple SNIs
}

func (e *SNIExtension) writeToUConn(uc *UConn) error {
	uc.config.ServerName = e.ServerName
	uc.HandshakeState.Hello.ServerName = e.ServerName
	return nil
}

func (e *SNIExtension) Len() int {
	return 4 + 2 + 1 + 2 + len(e.ServerName)
}

func (e *SNIExtension) Read(b []byte) (int, error) {
	if len(b) < e.Len() {
		return 0, io.ErrShortBuffer
	}
	// RFC 3546, section 3.1
	b[0] = byte(extensionServerName >> 8)
	b[1] = byte(extensionServerName)
	b[2] = byte((len(e.ServerName) + 5) >> 8)
	b[3] = byte((len(e.ServerName) + 5))
	b[4] = byte((len(e.ServerName) + 3) >> 8)
	b[5] = byte(len(e.ServerName) + 3)
	// b[6] Server Name Type: host_name (0)
	b[7] = byte(len(e.ServerName) >> 8)
	b[8] = byte(len(e.ServerName))
	copy(b[9:], []byte(e.ServerName))
	return e.Len(), io.EOF
}

type StatusRequestExtension struct {
}

func (e *StatusRequestExtension) writeToUConn(uc *UConn) error {
	uc.HandshakeState.Hello.OcspStapling = true
	return nil
}

func (e *StatusRequestExtension) Len() int {
	return 9
}

func (e *StatusRequestExtension) Read(b []byte) (int, error) {
	if len(b) < e.Len() {
		return 0, io.ErrShortBuffer
	}
	// RFC 4366, section 3.6
	b[0] = byte(extensionStatusRequest >> 8)
	b[1] = byte(extensionStatusRequest)
	b[2] = 0
	b[3] = 5
	b[4] = 1 // OCSP type
	// Two zero valued uint16s for the two lengths.
	return e.Len(), io.EOF
}

type SupportedCurvesExtension struct {
	Curves []CurveID
}

func (e *SupportedCurvesExtension) writeToUConn(uc *UConn) error {
	uc.config.CurvePreferences = e.Curves
	uc.HandshakeState.Hello.SupportedCurves = e.Curves
	return nil
}

func (e *SupportedCurvesExtension) Len() int {
	return 6 + 2*len(e.Curves)
}

func (e *SupportedCurvesExtension) Read(b []byte) (int, error) {
	if len(b) < e.Len() {
		return 0, io.ErrShortBuffer
	}
	// http://tools.ietf.org/html/rfc4492#section-5.5.1
	b[0] = byte(extensionSupportedCurves >> 8)
	b[1] = byte(extensionSupportedCurves)
	b[2] = byte((2 + 2*len(e.Curves)) >> 8)
	b[3] = byte((2 + 2*len(e.Curves)))
	b[4] = byte((2 * len(e.Curves)) >> 8)
	b[5] = byte((2 * len(e.Curves)))
	for i, curve := range e.Curves {
		b[6+2*i] = byte(curve >> 8)
		b[7+2*i] = byte(curve)
	}
	return e.Len(), io.EOF
}

type SupportedPointsExtension struct {
	SupportedPoints []uint8
}

func (e *SupportedPointsExtension) writeToUConn(uc *UConn) error {
	uc.HandshakeState.Hello.SupportedPoints = e.SupportedPoints
	return nil
}

func (e *SupportedPointsExtension) Len() int {
	return 5 + len(e.SupportedPoints)
}

func (e *SupportedPointsExtension) Read(b []byte) (int, error) {
	if len(b) < e.Len() {
		return 0, io.ErrShortBuffer
	}
	// http://tools.ietf.org/html/rfc4492#section-5.5.2
	b[0] = byte(extensionSupportedPoints >> 8)
	b[1] = byte(extensionSupportedPoints)
	b[2] = byte((1 + len(e.SupportedPoints)) >> 8)
	b[3] = byte((1 + len(e.SupportedPoints)))
	b[4] = byte((len(e.SupportedPoints)))
	for i, pointFormat := range e.SupportedPoints {
		b[5+i] = pointFormat
	}
	return e.Len(), io.EOF
}

type SignatureAlgorithmsExtension struct {
	SupportedSignatureAlgorithms []SignatureScheme
}

func (e *SignatureAlgorithmsExtension) writeToUConn(uc *UConn) error {
	uc.HandshakeState.Hello.SupportedSignatureAlgorithms = e.SupportedSignatureAlgorithms
	return nil
}

func (e *SignatureAlgorithmsExtension) Len() int {
	return 6 + 2*len(e.SupportedSignatureAlgorithms)
}

func (e *SignatureAlgorithmsExtension) Read(b []byte) (int, error) {
	if len(b) < e.Len() {
		return 0, io.ErrShortBuffer
	}
	// https://tools.ietf.org/html/rfc5246#section-7.4.1.4.1
	b[0] = byte(extensionSignatureAlgorithms >> 8)
	b[1] = byte(extensionSignatureAlgorithms)
	b[2] = byte((2 + 2*len(e.SupportedSignatureAlgorithms)) >> 8)
	b[3] = byte((2 + 2*len(e.SupportedSignatureAlgorithms)))
	b[4] = byte((2 * len(e.SupportedSignatureAlgorithms)) >> 8)
	b[5] = byte((2 * len(e.SupportedSignatureAlgorithms)))
	for i, sigAndHash := range e.SupportedSignatureAlgorithms {
		b[6+2*i] = byte(sigAndHash >> 8)
		b[7+2*i] = byte(sigAndHash)
	}
	return e.Len(), io.EOF
}

type RenegotiationInfoExtension struct {
	renegotiation       RenegotiationSupport
	SecureRenegotiation []byte // you probably want to leave it empty
}

func (e *RenegotiationInfoExtension) writeToUConn(uc *UConn) error {
	uc.config.Renegotiation = e.renegotiation
	switch e.renegotiation {
	case RenegotiateOnceAsClient:
		fallthrough
	case RenegotiateFreelyAsClient:
		uc.HandshakeState.Hello.SecureRenegotiationSupported = true
		// Note that if we manage to use this in renegotiation(currently only in initial handshake), we'd have to point
		// uc.HandshakeState.Hello.SecureRenegotiation = chs.C.clientFinished
		// and probably do something else. It's a mess.
	case RenegotiateNever:
	default:
	}
	return nil
}

func (e *RenegotiationInfoExtension) Len() int {
	switch e.renegotiation {
	case RenegotiateOnceAsClient:
		fallthrough
	case RenegotiateFreelyAsClient:
		return 5 + len(e.SecureRenegotiation)
	case RenegotiateNever:
	default:
	}
	return 0
}

func (e *RenegotiationInfoExtension) Read(b []byte) (int, error) {
	if len(b) < e.Len() {
		return 0, io.ErrShortBuffer
	}
	switch e.renegotiation {
	case RenegotiateOnceAsClient:
		fallthrough
	case RenegotiateFreelyAsClient:
		b[0] = byte(extensionRenegotiationInfo >> 8)
		b[1] = byte(extensionRenegotiationInfo & 0xff)
		b[2] = 0 // TODO: this is not what Chrome does :(
		b[3] = byte(len(e.SecureRenegotiation) + 1)
		b[4] = byte(len(e.SecureRenegotiation))
		if len(e.SecureRenegotiation) != 0 {
			copy(b[5:], e.SecureRenegotiation)
		}
	case RenegotiateNever:
	default:
	}
	return e.Len(), io.EOF
}

type ALPNExtension struct {
	AlpnProtocols []string
}

func (e *ALPNExtension) writeToUConn(uc *UConn) error {
	uc.config.NextProtos = e.AlpnProtocols
	uc.HandshakeState.Hello.AlpnProtocols = e.AlpnProtocols
	return nil
}

func (e *ALPNExtension) Len() int {
	bLen := 2 + 2 + 2
	for _, s := range e.AlpnProtocols {
		bLen += 1 + len(s)
	}
	return bLen
}

func (e *ALPNExtension) Read(b []byte) (int, error) {
	if len(b) < e.Len() {
		return 0, io.ErrShortBuffer
	}

	b[0] = byte(extensionALPN >> 8)
	b[1] = byte(extensionALPN & 0xff)
	lengths := b[2:]
	b = b[6:]

	stringsLength := 0
	for _, s := range e.AlpnProtocols {
		l := len(s)
		b[0] = byte(l)
		copy(b[1:], s)
		b = b[1+l:]
		stringsLength += 1 + l
	}

	lengths[2] = byte(stringsLength >> 8)
	lengths[3] = byte(stringsLength)
	stringsLength += 2
	lengths[0] = byte(stringsLength >> 8)
	lengths[1] = byte(stringsLength)

	return e.Len(), io.EOF
}

type SCTExtension struct {
}

func (e *SCTExtension) writeToUConn(uc *UConn) error {
	uc.HandshakeState.Hello.Scts = true
	return nil
}

func (e *SCTExtension) Len() int {
	return 4
}

func (e *SCTExtension) Read(b []byte) (int, error) {
	if len(b) < e.Len() {
		return 0, io.ErrShortBuffer
	}
	// https://tools.ietf.org/html/rfc6962#section-3.3.1
	b[0] = byte(extensionSCT >> 8)
	b[1] = byte(extensionSCT)
	// zero uint16 for the zero-length extension_data
	return e.Len(), io.EOF
}

type SessionTicketExtension struct {
	Session *ClientSessionState
}

func (e *SessionTicketExtension) writeToUConn(uc *UConn) error {
	if e.Session != nil {
		uc.HandshakeState.Session = e.Session
		uc.HandshakeState.Hello.SessionTicket = e.Session.sessionTicket
	}
	return nil
}

func (e *SessionTicketExtension) Len() int {
	if e.Session != nil {
		return 4 + len(e.Session.sessionTicket)
	}
	return 4
}

func (e *SessionTicketExtension) Read(b []byte) (int, error) {
	if len(b) < e.Len() {
		return 0, io.ErrShortBuffer
	}

	extBodyLen := e.Len() - 4

	b[0] = byte(extensionSessionTicket >> 8)
	b[1] = byte(extensionSessionTicket)
	b[2] = byte(extBodyLen >> 8)
	b[3] = byte(extBodyLen)
	if extBodyLen > 0 {
		copy(b[4:], e.Session.sessionTicket)
	}
	return e.Len(), io.EOF
}

/*
FAKE EXTENSIONS
*/

type FakeChannelIDExtension struct {
}

func (e *FakeChannelIDExtension) writeToUConn(uc *UConn) error {
	return nil
}

func (e *FakeChannelIDExtension) Len() int {
	return 4
}

func (e *FakeChannelIDExtension) Read(b []byte) (int, error) {
	if len(b) < e.Len() {
		return 0, io.ErrShortBuffer
	}
	// https://tools.ietf.org/html/draft-balfanz-tls-channelid-00
	b[0] = byte(fakeExtensionChannelID >> 8)
	b[1] = byte(fakeExtensionChannelID & 0xff)
	// The length is 0
	return e.Len(), io.EOF
}

type utlsExtendedMasterSecretExtension struct {
}

// TODO: update when this extension is implemented in crypto/tls
// but we probably won't have to enable it in Config
func (e *utlsExtendedMasterSecretExtension) writeToUConn(uc *UConn) error {
	uc.HandshakeState.Hello.Ems = true
	return nil
}

func (e *utlsExtendedMasterSecretExtension) Len() int {
	return 4
}

func (e *utlsExtendedMasterSecretExtension) Read(b []byte) (int, error) {
	if len(b) < e.Len() {
		return 0, io.ErrShortBuffer
	}
	// https://tools.ietf.org/html/rfc7627
	b[0] = byte(utlsExtensionExtendedMasterSecret >> 8)
	b[1] = byte(utlsExtensionExtendedMasterSecret)
	// The length is 0
	return e.Len(), io.EOF
}

var extendedMasterSecretLabel = []byte("extended master secret")

// extendedMasterFromPreMasterSecret generates the master secret from the pre-master
// secret and session hash. See https://tools.ietf.org/html/rfc7627#section-4
func extendedMasterFromPreMasterSecret(version uint16, suite *cipherSuite, preMasterSecret []byte, fh finishedHash) []byte {
	sessionHash := fh.Sum()
	masterSecret := make([]byte, masterSecretLength)
	prfForVersion(version, suite)(masterSecret, preMasterSecret, extendedMasterSecretLabel, sessionHash)
	return masterSecret
}

// GREASE stinks with dead parrots, have to be super careful, and, if possible, not include GREASE
const (
	ssl_grease_cipher = iota
	ssl_grease_group
	ssl_grease_extension1
	ssl_grease_extension2
	ssl_grease_version
	ssl_grease_ticket_extension
)

// it is responsibility of user not to generate multiple grease extensions with same value
type FakeGREASEExtension struct {
	Value uint16
	Body  []byte // in Chrome first grease has empty body, second grease has a single zero byte
}

func (e *FakeGREASEExtension) writeToUConn(uc *UConn) error {
	return nil
}

// will panic if clientRandom[index] is out of bounds.
func GetBoringGREASEValue(clientRandom []byte, index int) uint16 {
	// Get GREASE value BoringSSL-style. Unfortunately, this value isn't really boring and is quite interesting :(
	// https://github.com/google/boringssl/blob/a365138ac60f38b64bfc608b493e0f879845cb88/ssl/handshake_client.c#L530
	ret := uint16(clientRandom[index])
	/* This generates a random value of the form 0xωaωa, for all 0 ≤ ω < 16. */
	ret = (ret & 0xf0) | 0x0a
	ret |= ret << 8
	return ret
}

func (e *FakeGREASEExtension) Len() int {
	return 4 + len(e.Body)
}

func (e *FakeGREASEExtension) Read(b []byte) (int, error) {
	if len(b) < e.Len() {
		return 0, io.ErrShortBuffer
	}

	b[0] = byte(e.Value >> 8)
	b[1] = byte(e.Value)
	b[2] = byte(len(e.Body) >> 8)
	b[3] = byte(len(e.Body))
	if len(e.Body) > 0 {
		copy(b[4:], e.Body)
	}
	return e.Len(), io.EOF
}

//
type utlsPaddingExtension struct {
	PaddingLen int
	WillPad    bool // set to false to disable extension

	// Functor for deciding on padding length based on unpadded ClientHello length.
	// If willPad is false, then this extension should not be included.
	GetPaddingLen func(clientHelloUnpaddedLen int) (paddingLen int, willPad bool)
}

func (e *utlsPaddingExtension) writeToUConn(uc *UConn) error {
	return nil
}

func (e *utlsPaddingExtension) Len() int {
	if e.WillPad {
		return 4 + e.PaddingLen
	} else {
		return 0
	}
}

func (e *utlsPaddingExtension) Update(clientHelloUnpaddedLen int) {
	if e.GetPaddingLen != nil {
		e.PaddingLen, e.WillPad = e.GetPaddingLen(clientHelloUnpaddedLen)
	}
}

func (e *utlsPaddingExtension) Read(b []byte) (int, error) {
	if !e.WillPad {
		return 0, io.EOF
	}
	if len(b) < e.Len() {
		return 0, io.ErrShortBuffer
	}
	// https://tools.ietf.org/html/rfc7627
	b[0] = byte(utlsExtensionPadding >> 8)
	b[1] = byte(utlsExtensionPadding)
	b[2] = byte(e.PaddingLen >> 8)
	b[3] = byte(e.PaddingLen)
	return e.Len(), io.EOF
}

// https://github.com/google/boringssl/blob/7d7554b6b3c79e707e25521e61e066ce2b996e4c/ssl/t1_lib.c#L2803
func boringPaddingStyle(unpaddedLen int) (int, bool) {
	if unpaddedLen > 0xff && unpaddedLen < 0x200 {
		paddingLen := 0x200 - unpaddedLen
		if paddingLen >= 4+1 {
			paddingLen -= 4
		} else {
			paddingLen = 1
		}
		return paddingLen, true
	}
	return 0, false
}
