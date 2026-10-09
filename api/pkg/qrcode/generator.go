// Package qrcode renders text, such as a client configuration, as a QR code
// that the AmneziaVPN app can scan.
package qrcode

import (
	"encoding/base64"
	"fmt"

	qrc "github.com/skip2/go-qrcode"
)

// GeneratePNGDataURL returns content as a PNG QR code of size pixels, encoded
// as a data URL so the UI can put it into an img tag without another request.
func GeneratePNGDataURL(content string, size int) (string, error) {
	png, err := qrc.Encode(content, qrc.Medium, size)
	if err != nil {
		return "", fmt.Errorf("encode QR code: %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}
