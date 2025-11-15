package newmessage

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

const AwaitMessageText = "Получил твоё сообщение! Скоро отвечу ;3"

func NewMessageChain() *handlers.HandlerChain {
	return handlers.HandlerChain{}.Init(
		10*time.Second,
		shared.GetOrCrateThread,
		handlers.InitChainHandler(CheckUserIsBlocked),
		handlers.InitChainHandler(SaveResentMessage),
		handlers.InitChainHandler(RedirectMessageToThread),
		handlers.InitChainHandler(UpdateResentMessage),
	)
}

func CheckUserIsBlocked(c tele.Context, args *handlers.Arg) (*handlers.Arg, *e.ErrorInfo) {
	user := (*args)["user"].(*models.User)
	if user.IsBlocked {
		c.Reply("You are blocked. There is nothting you can do with it :3")
		return args, e.NewError("user is blocked", "User is blocked").WithSeverity(e.Ingnored).WithData(map[string]any{
			"user": user,
		})
	}

	return args, e.Nil()
}

func SaveResentMessage(c tele.Context, args *handlers.Arg) (*handlers.Arg, *e.ErrorInfo) {
	thread := (*args)["thread"].(*models.Thread)

	var SenderChatMessageID, TargetChatMessageID int
	if (*args)["sender_is_owner"].(bool) {
		SenderChatMessageID = -1
		TargetChatMessageID = c.Message().ID
	} else {
		SenderChatMessageID = c.Message().ID
		TargetChatMessageID = -1
	}

	resentMessage := &models.ResentMessage{
		ThreadID: thread.ID,
		ChatID: thread.ChatID,
		SenderChatMessageID: SenderChatMessageID,
		TargetChatMessageID: TargetChatMessageID,
	}

	_, err := database.GetDB().Model(resentMessage).Insert()
	if err != nil {
		return args, e.FromError(err, "Failed to insert resent message").WithSeverity(e.Notice).WithData(map[string]any{
			"resent_message": resentMessage,
		})
	}

	(*args)["resent_message"] = resentMessage

	return args, e.Nil()
}

func RedirectMessageToThread(c tele.Context, args *handlers.Arg) (*handlers.Arg, *e.ErrorInfo) {
	if (*args)["user"].(*models.User).IsOwner {
		return RedirectFromThreadToUser(c, args)
	}

	return RedirectFromUserToThread(c, args)
}

func RedirectFromThreadToUser(c tele.Context, args *handlers.Arg) (*handlers.Arg, *e.ErrorInfo) {
	thread := (*args)["thread"].(*models.Thread)
	
	chatRecipient := &tele.Chat{ID: thread.AssociatedUserID}
	options := &tele.SendOptions{}
	
	if c.Message().ReplyTo != nil {
		replyToMessageID := getReplyToMessageID((*args)["user"].(*models.User), c.Message().ReplyTo.ID, thread, database.GetDB())
		options.ReplyTo = &tele.Message{ID: replyToMessageID}
	}

	msg, err := c.Bot().Copy(
		chatRecipient,
		c.Message(),
		options,
	)
	if err != nil {
		return args, e.FromError(err, "Failed to copy message").WithSeverity(e.Notice).WithData(map[string]any{
			"message": msg,
		})
	}

	(*args)["sent_message"] = msg

	return args, e.Nil()
}

func RedirectFromUserToThread(c tele.Context, args *handlers.Arg) (*handlers.Arg, *e.ErrorInfo) {
	thread := (*args)["thread"].(*models.Thread)

	chatRecipient := &tele.Chat{ID: thread.ChatID, Type: tele.ChatSuperGroup}
	options := &tele.SendOptions{ThreadID: thread.ThreadID}
	
	if c.Message().ReplyTo != nil {
		replyToMessageID := getReplyToMessageID((*args)["user"].(*models.User), c.Message().ReplyTo.ID, thread, database.GetDB())
		options.ReplyTo = &tele.Message{ID: replyToMessageID}
	}

	msg, err := c.Bot().Copy(
		chatRecipient,
		c.Message(),
		options,
	)
	if err != nil {
		return args, e.FromError(err, "Failed to copy message").WithSeverity(e.Notice).WithData(map[string]any{
			"message": msg,
		})
	}

	(*args)["sent_message"] = msg

	c.Send(AwaitMessageText)

	return args, e.Nil()
}

func getReplyToMessageID(user *models.User, replyToMessageID int, thread *models.Thread, db *pg.DB) int {		
	if user.IsOwner {
		var message = &models.ResentMessage{}
		err := db.Model(message).
			Where("thread_id = ?", thread.ID).
			Where("chat_id = ?", thread.ChatID).
			Where("target_chat_message_id = ?", replyToMessageID).
			Select()
		if err != nil {
			fmt.Println("Error selecting resent message: ", err)
			return 0
		}
		return message.SenderChatMessageID
	}
	
	var message = &models.ResentMessage{}
	
	err := db.Model(message).
		Where("thread_id = ?", thread.ID).
		Where("chat_id = ?", thread.ChatID).
		Where("sender_chat_message_id = ?", replyToMessageID).
		Select()
	if err != nil {
		fmt.Println("Error selecting resent message: ", err)
		return 0
	}

	return message.TargetChatMessageID
}

func UpdateResentMessage(c tele.Context, args *handlers.Arg) (*handlers.Arg, *e.ErrorInfo) {
	resentMessage := (*args)["resent_message"].(*models.ResentMessage)
	if (*args)["user"].(*models.User).IsOwner {
		resentMessage.SenderChatMessageID = (*args)["sent_message"].(*tele.Message).ID
	} else {
		resentMessage.TargetChatMessageID = (*args)["sent_message"].(*tele.Message).ID
	}

	_, err := database.GetDB().Model(resentMessage).WherePK().Column("target_chat_message_id", "sender_chat_message_id").Update()
	if err != nil {
		return args, e.FromError(err, "Failed to update resent message").WithSeverity(e.Notice).WithData(map[string]any{
			"resent_message": resentMessage,
		})
	}

	return args, e.Nil()
}
