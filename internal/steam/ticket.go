// Package steam validates Steam auth-session tickets and extracts the
// entitlements they carry; a port of Peacock's SteamStrategy and parseAppTicket.
package steam

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// steamPublicKeyPEM signs app ownership tickets.
const steamPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIGdMA0GCSqGSIb3DQEBAQUAA4GLADCBhwKBgQDf7BrWLBBmLBc1OhSwfFkRf53T
2Ct64+AVzRkeRuh7h3SiGEYxqQMUeYKO6UWiSRKpI2hzic9pobFhRr3Bvr/WARvY
gdTckPv+T1JzZsuVcNfFjrocejN1oWI0Rrtgt4Bo+hOneoo3S57G9F1fOpn5nsQ6
6WOiu4gZKODnFMBCiQIBEQ==
-----END PUBLIC KEY-----`

var steamPublicKey = mustParsePublicKey(steamPublicKeyPEM)

func mustParsePublicKey(p string) *rsa.PublicKey {
	block, _ := pem.Decode([]byte(p))
	if block == nil {
		panic("steam: invalid public key PEM")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(fmt.Sprintf("steam: parse public key: %v", err))
	}
	return key.(*rsa.PublicKey)
}

const signatureSize = 128

type AppTicket struct {
	SteamId string
	AppId   string
	Expiry  time.Time
	DLC     []string // excludes AppId
	Hash    string   // SHA-256 of the signed section, used as the cache key
	Valid   bool     // signature verified and not expired
}

// Entitlements returns AppId followed by the owned DLC ids.
func (t *AppTicket) Entitlements() []string {
	return append([]string{t.AppId}, t.DLC...)
}

// ParseAppTicket decodes a hex steam_clienttoken and verifies its signature.
func ParseAppTicket(hexTicket string) (*AppTicket, error) {
	raw, err := hex.DecodeString(hexTicket)
	if err != nil {
		return nil, fmt.Errorf("decode ticket hex: %w", err)
	}
	return parseAppTicket(raw, steamPublicKey, time.Now())
}

var errTruncated = errors.New("ticket truncated")

// ticketReader is a bounds-checked little-endian cursor; the first error sticks.
type ticketReader struct {
	buf []byte
	off int
	err error
}

func (r *ticketReader) need(n int) bool {
	if r.err != nil {
		return false
	}
	if n < 0 || r.off+n > len(r.buf) {
		r.err = errTruncated
		return false
	}
	return true
}

func (r *ticketReader) skip(n int) {
	if r.need(n) {
		r.off += n
	}
}

func (r *ticketReader) u16() uint16 {
	if !r.need(2) {
		return 0
	}
	v := binary.LittleEndian.Uint16(r.buf[r.off:])
	r.off += 2
	return v
}

func (r *ticketReader) u32() uint32 {
	if !r.need(4) {
		return 0
	}
	v := binary.LittleEndian.Uint32(r.buf[r.off:])
	r.off += 4
	return v
}

func (r *ticketReader) u64() uint64 {
	if !r.need(8) {
		return 0
	}
	v := binary.LittleEndian.Uint64(r.buf[r.off:])
	r.off += 8
	return v
}

func (r *ticketReader) skipLicenses() {
	n := r.u16()
	r.skip(int(n) * 4)
}

// parseAppTicket follows SteamKit's steam3_appticket.hsl layout.
func parseAppTicket(raw []byte, pub *rsa.PublicKey, now time.Time) (*AppTicket, error) {
	r := &ticketReader{buf: raw}

	// Skip the GC token and session header in front of the ownership ticket.
	if len(raw) >= 4 && binary.LittleEndian.Uint32(raw) == 20 {
		r.skip(52 + 4)
	}

	start := r.off
	length := int(r.u32())
	version := r.u32()
	if r.err != nil {
		return nil, r.err
	}
	if version != 4 {
		return nil, fmt.Errorf("unknown ownership ticket version %d (want 4)", version)
	}
	if !r.need(length - 8) {
		return nil, r.err
	}
	signed := raw[start : start+length]

	t := &AppTicket{}
	t.SteamId = strconv.FormatUint(r.u64(), 10)
	t.AppId = strconv.FormatUint(uint64(r.u32()), 10)
	r.skip(16) // external IP, internal IP, ownership flags, time issued
	t.Expiry = time.Unix(int64(r.u32()), 0)

	r.skipLicenses()

	dlcCount := r.u16()
	for i := 0; i < int(dlcCount) && r.err == nil; i++ {
		t.DLC = append(t.DLC, strconv.FormatUint(uint64(r.u32()), 10))
		r.skipLicenses()
	}

	r.skip(2) // reserved
	if r.err != nil {
		return nil, r.err
	}

	sum := sha256.Sum256(signed)
	t.Hash = hex.EncodeToString(sum[:])

	validSignature := false
	if r.off+signatureSize == len(raw) {
		digest := sha1.Sum(signed)
		validSignature = rsa.VerifyPKCS1v15(pub, crypto.SHA1, digest[:], raw[r.off:]) == nil
	}

	t.Valid = validSignature && now.Before(t.Expiry)
	return t, nil
}
