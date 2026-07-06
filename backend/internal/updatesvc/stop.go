package updatesvc

import (
	"context"
	"errors"
)

var ErrStopped = errors.New("更新已停止")

func stopIfCanceled(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	err := ctx.Err()
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return ErrStopped
	}
	return err
}

func IsStopped(err error) bool {
	return errors.Is(err, ErrStopped)
}
