package utils

import "go.uber.org/zap"

func SafeSend(ch chan<- any, msg any, logger *zap.SugaredLogger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Warnf("send failed (likely closed channel): %v", r)
		}
	}()

	ch <- msg
}
