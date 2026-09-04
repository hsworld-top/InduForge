package service

import (
	"fmt"
	"strconv"
	"strings"
)

func FormatAuthoringEpoch(epoch int64) string { return "epoch-" + strconv.FormatInt(epoch, 10) }

func ParseAuthoringEpoch(value string) (int64, error) {
	if !strings.HasPrefix(value, "epoch-") {
		return 0, fmt.Errorf("工程开发态代次无效")
	}
	epoch, err := strconv.ParseInt(strings.TrimPrefix(value, "epoch-"), 10, 64)
	if err != nil || epoch < 1 {
		return 0, fmt.Errorf("工程开发态代次无效")
	}
	return epoch, nil
}
