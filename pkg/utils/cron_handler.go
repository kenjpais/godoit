package utils

import (
	"github.com/robfig/cron/v3"
	"fmt"
    "time"
)

func EvalCronExpr(cronExpr string) (time.Time, error) {
    c, err := cron.ParseStandard(cronExpr)
    if err != nil {
        return time.Time{}, fmt.Errorf("invalid cron expression: %v", err)
    }
    return c.Next(time.Now()), nil
}