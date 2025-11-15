package commandstart

import (
	"app/internal/handlers"
	"app/internal/handlers/shared"
	e "app/pkg/errors"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"
)

func CommandStartChain() *handlers.HandlerChain {
	return handlers.HandlerChain{}.Init(
		10*time.Second,
		shared.GetOrCrateThread,
		handlers.InitChainHandler(SendGreetingMessage),
	)
}


func SendGreetingMessage(c tele.Context, args *handlers.Arg) (*handlers.Arg, *e.ErrorInfo) {
	err := c.Reply(fmt.Sprintf("Привет, %s!\n\nЭта штука работает как обычный чат: каждое сообщение отправленное в бота будет доставлено эдмину бота; его ответы будут приходить сюда же.\nПоддерживаются все типы медиа, а также стикеры, видеосообщения, голосовые итд. Также бот поддерживает реплаи (ответы на сообщения)\n\nЗачем это сделано?\nДанный бот призван максимально минимизировать неудобства при общении при этом оставив меня анонимным. Это обеспечивает мою безопасность ;)", c.Sender().FirstName))
	if err != nil {
		return args, e.FromError(err, "Failed to send greeting message").WithSeverity(e.Critical).WithData(map[string]any{
			"user": (*args)["user"],
		})
	}
	c.Send("-*- Для того чтобы начать просто отправь любое сообщение -*-")

	return args, e.Nil()
}
