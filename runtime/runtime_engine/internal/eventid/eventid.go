// Package eventid 实现 Runtime Data Plane V1 的确定性事件标识与原始 body 摘要。
package eventid

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const separator = '\x1f'

// HashFields 以契约规定的 0x1f 分隔符计算 SHA-256；分隔符出现在输入中会造成字段歧义，必须拒绝。
func HashFields(fields ...string) (string, error) {
	for _, field := range fields {
		if strings.ContainsRune(field, separator) {
			return "", fmt.Errorf("eventId 输入字段含有非法 0x1f 分隔符")
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(fields, string(separator))))
	return hex.EncodeToString(sum[:]), nil
}

func Raw(schemaVersion, deploymentID, pointID, ownerID string, epoch, sequence int64) (string, error) {
	return HashFields(schemaVersion, deploymentID, pointID, ownerID, strconv.FormatInt(epoch, 10), strconv.FormatInt(sequence, 10))
}

func Computed(schemaVersion, deploymentID, pointID, ownerID string, epoch, sequence int64, computeID string, revision int64, inputEventIDs []string) (string, error) {
	inputs := append([]string(nil), inputEventIDs...)
	sort.Strings(inputs)
	fields := []string{schemaVersion, deploymentID, pointID, ownerID, strconv.FormatInt(epoch, 10), strconv.FormatInt(sequence, 10), computeID, strconv.FormatInt(revision, 10)}
	return HashFields(append(fields, inputs...)...)
}

func AlarmTransition(schemaVersion, kind, deploymentID, alarmID, operation, ownerID string, epoch int64, sourceTimestamp, alarmItemID string, pointIDs []string, stateVersion, transitionSequence int64) (string, error) {
	points := append([]string(nil), pointIDs...)
	sort.Strings(points)
	fields := []string{schemaVersion, kind, deploymentID, alarmID, operation, ownerID, strconv.FormatInt(epoch, 10), sourceTimestamp, alarmItemID}
	fields = append(fields, points...)
	fields = append(fields, strconv.FormatInt(stateVersion, 10), strconv.FormatInt(transitionSequence, 10))
	return HashFields(fields...)
}

func DataGap(schemaVersion, kind, deploymentID, collectorID, connectionID, ownerID string, epoch, fromSequence, toSequence int64, detectedAt, reason string) (string, error) {
	return HashFields(schemaVersion, kind, deploymentID, collectorID, connectionID, ownerID, strconv.FormatInt(epoch, 10), strconv.FormatInt(fromSequence, 10), strconv.FormatInt(toSequence, 10), detectedAt, reason)
}

// BodySHA256 是 JetStream 接收的原始 payload bytes 的摘要；调用方不得先反序列化再序列化。
func BodySHA256(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
