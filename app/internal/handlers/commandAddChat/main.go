package commandaddchat

import (
	"app/internal/handlers"
	"app/internal/handlers/shared"
	"fmt"
	"time"

	"app/pkg/database"
	"app/pkg/database/models"
	e "app/pkg/errors"

	"github.com/go-pg/pg/v10"
	tele "gopkg.in/telebot.v4"
)

func CommandAddChatChain() *handlers.HandlerChain {
	return handlers.HandlerChain{}.Init(
		10*time.Second,
		shared.GetSenderAndTargetUser,
		handlers.InitChainHandler(InitNewChatForUser),
		handlers.InitChainHandler(SendChatAddedMessage),
	)
}

func InitNewChatForUser(c tele.Context, args *handlers.Arg) (*handlers.Arg, *e.ErrorInfo) {
	db := database.GetDB()

	if c.Chat().Type != tele.ChatSuperGroup {
		c.Reply("Chat should be a supergroup")
		return args, e.NewError("chat should be a supergroup", "Chat should be a supergroup").WithSeverity(e.Ingnored).WithData(map[string]any{
			"user": (*args)["user"],
		})
	}
	
	var chat models.Chat
	err := db.Model(&chat).
		Where("chat_owner_id = ?", (*args)["user"].(*models.User).TgID).
		Select()

	if err == nil {
		c.Reply("Chat already exists. Replace with new one? (TODO: Implement)")
		return args, e.NewError("chat already exists", "Chat already exists").WithSeverity(e.Ingnored).WithData(map[string]any{
			"user": (*args)["user"],
			"chat": chat,
		})
	}
	
	if err != pg.ErrNoRows {
		return args, e.FromError(err, "Failed to select chat").WithSeverity(e.Notice).WithData(map[string]any{
			"user": (*args)["user"],
		})
	}

	chat = models.Chat{
		ChatOwnerID: (*args)["user"].(*models.User).TgID,
		TgID: c.Chat().ID,
	}
	_, err = db.Model(&chat).Insert()
	if err != nil {
		return args, e.FromError(err, "Failed to insert chat").WithSeverity(e.Notice).WithData(map[string]any{
			"chat": chat,
		})
	}

	return args, e.Nil()
}

func SendChatAddedMessage(c tele.Context, args *handlers.Arg) (*handlers.Arg, *e.ErrorInfo) {
	c.Reply(fmt.Sprintf("Chat %s added successfully", c.Chat().Title))
	return args, e.Nil()
}