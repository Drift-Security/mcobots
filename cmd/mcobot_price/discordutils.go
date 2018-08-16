package main

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/stroncium/gg"
)

func DiscordPNGData(img *gg.Context) (data string, err error) {
	var buf bytes.Buffer
	if err = img.EncodePNG(&buf); err != nil {
		return
	}
	data = DiscordPNGFromBytes(buf.Bytes())
	return
}

func DiscordPNGFromBytes(bytes []byte) string {
	return fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(bytes))
}
