package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/ccheers/xpkg/xmsgbus"
)

var errRecoveryUnavailable = errors.New("恢复暂时无法推进")

func (x *MsgBus) claim(ctx context.Context, topic, channel, receipt, generation string) ([]byte, bool, error) {
	keys := channelKeysFor(topic, channel)
	result, err := x.client.Eval(ctx, claimScript,
		[]string{pendingDataKey, pendingDueKey, keys.catalog, keys.ready}, channel, receipt, generation)
	if err != nil {
		return nil, false, fmt.Errorf("claim message: %w", err)
	}
	values, ok := result.([]interface{})
	if !ok || len(values) > 2 {
		return nil, false, fmt.Errorf("claim message: unexpected result %T", result)
	}
	if len(values) == 0 {
		return nil, false, nil
	}
	if len(values) == 2 {
		return nil, false, xmsgbus.ErrNoData
	}
	data, err := ackResultString(values[0])
	if err != nil {
		return nil, false, fmt.Errorf("claim message: %w", err)
	}
	return []byte(data), true, nil
}

func (x *MsgBus) acknowledge(ctx context.Context, receipt string) error {
	_, err := x.client.Eval(ctx, acknowledgeScript,
		[]string{pendingDataKey, pendingDueKey}, receipt)
	if err != nil {
		return fmt.Errorf("acknowledge message: %w", err)
	}
	return nil
}

// Recover 将至多 limit 条到期回执对应的消息重新入队，返回实际入队数量。
func (x *MsgBus) Recover(ctx context.Context, limit int) (int, error) {
	if limit < 1 || limit > 1000 {
		return 0, fmt.Errorf("recover messages: limit must be between 1 and 1000")
	}
	candidates, err := x.pendingCandidates(ctx, limit)
	if err != nil {
		return 0, err
	}
	recovered := 0
	var errs []error
	for _, candidate := range candidates {
		receipt, err := ackResultString(candidate)
		if err != nil {
			return recovered, fmt.Errorf("read pending receipt: %w", err)
		}
		result, err := x.client.Eval(ctx, recoverScript,
			[]string{pendingDataKey, pendingDueKey}, receipt)
		if err != nil {
			errs = append(errs, fmt.Errorf("recover receipt %q: %w", receipt, err))
			// 保留坏记录并延后重试，不能让它永远占着全局到期队列的队首。
			if _, deferErr := x.client.Eval(ctx, deferRecoveryScript, []string{pendingDueKey}, receipt); deferErr != nil {
				return recovered, errors.Join(append(errs, deferErr, errRecoveryUnavailable)...)
			}
			continue
		}
		count, ok := result.(int64)
		if !ok || (count != 0 && count != 1) {
			return recovered, fmt.Errorf("recover receipt %q: unexpected result %v", receipt, result)
		}
		recovered += int(count)
	}
	return recovered, errors.Join(errs...)
}

func (x *MsgBus) pendingCandidates(ctx context.Context, limit int) ([]interface{}, error) {
	result, err := x.client.Eval(ctx, pendingCandidatesScript,
		[]string{pendingDataKey, pendingDueKey}, limit)
	if err != nil {
		return nil, errors.Join(errRecoveryUnavailable, fmt.Errorf("list pending messages: %w", err))
	}
	candidates, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("list pending messages: unexpected result %T", result)
	}
	return candidates, nil
}

func ackResultString(value interface{}) (string, error) {
	switch value := value.(type) {
	case string:
		return value, nil
	case []byte:
		return string(value), nil
	default:
		return "", fmt.Errorf("unexpected Redis string result %T", value)
	}
}
