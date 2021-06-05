package git

import (
	"encoding/base64"
	"io"
)

// returns true if error is an end-of-file error
func eof(err error) bool { return err == io.EOF }

func enc(buf []byte) string          { return base64.StdEncoding.EncodeToString(buf) }
func dec(str string) ([]byte, error) { return base64.StdEncoding.DecodeString(str) }
