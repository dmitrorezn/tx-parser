package converter

import (
	"errors"
	"strconv"
)

const (
	hexPrefix          = "0x"
	hexBase            = 16
	blockNumberBitSize = 64
)

var ErrInvalidHexValueLen = errors.New("invalid hex number value  len")

func ParseHexInt(str string) (int, error) {
	if len(str) < len(hexPrefix) {
		return 0, ErrInvalidHexValueLen
	}
	number, err := strconv.ParseInt(str[len(hexPrefix):], hexBase, blockNumberBitSize)
	if err != nil {
		return 0, err
	}

	return int(number), err
}

func FormatHexInt(i int) string {
	return hexPrefix + strconv.FormatInt(int64(i), hexBase)
}
