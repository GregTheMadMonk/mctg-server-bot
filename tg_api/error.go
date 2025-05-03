package tg_api

import (
    "fmt"
)

func (self *ExchangeResult[T]) Error() string {
    return fmt.Sprintf(
        "Telegram API error code %d: %s",
        self.ErrorCode,
        self.Description,
    )
} // <-- *ExchangeResult[T]::Error()
