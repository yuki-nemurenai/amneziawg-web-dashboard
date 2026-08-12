package qrcode

import (
	"encoding/base64"
	"fmt"

	qrc "github.com/skip2/go-qrcode"
)

// GeneratePNGDataURL returns a base64 encoded data URL PNG string of the content
func GeneratePNGDataURL(content string, size int) (string, error) {
	png, err := qrc.Encode(content, qrc.Medium, size)
	if err != nil {
		return "", fmt.Errorf("failed to encode qr code: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(png)
	return fmt.Sprintf("data:image/png;base64,%s", encoded), nil
}

// GeneratePNGBytes returns raw PNG bytes for downloading
func GeneratePNGBytes(content string, size int) ([]byte, error) {
	png, err := qrc.Encode(content, qrc.Medium, size)
	if err != nil {
		return nil, fmt.Errorf("failed to encode qr code: %w", err)
	}
	return png, nil
}
