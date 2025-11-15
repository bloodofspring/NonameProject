package shared

import (
	"app/internal/handlers"
	"app/pkg/database"
	"app/pkg/database/models"
	e "app/pkg/errors"
	"fmt"
	"math/rand"

	"github.com/go-pg/pg/v10"
	"github.com/spf13/viper"
	tele "gopkg.in/telebot.v4"
)

func getSenderFunc(c tele.Context, args *handlers.Arg) (*handlers.Arg, *e.ErrorInfo) {
	db := database.GetDB()

	user := &models.User{
		TgID: c.Sender().ID,
	}

	_, err := db.Model(user).WherePK().SelectOrInsert()
	if err != nil {
		return args, e.FromError(err, "Failed to insert user").WithSeverity(e.Notice)
	}

	(*args)["user"] = user

	return args, e.Nil()
}

func getTargetFunc(c tele.Context, args *handlers.Arg) (*handlers.Arg, *e.ErrorInfo) {
	db := database.GetDB()

	target := &models.User{
		TgID: viper.GetInt64("OWNER_TG_ID"),
		IsOwner: true,
	}

	_, err := db.Model(target).WherePK().Returning("*").SelectOrInsert()
	if err != nil {
		return args, e.FromError(err, "Failed to select target user").WithSeverity(e.Notice).WithData(map[string]any{
			"target": target,
		})
	}

	(*args)["target"] = target

	return args, e.Nil()
}

func getSenderAndTargetUserFunc(c tele.Context, args *handlers.Arg) (*handlers.Arg, *e.ErrorInfo) {
	(*args)["sender_is_owner"] = (*args)["user"].(*models.User).IsOwner

	return args, e.Nil()
}

func getThread(threadID int, chatID int64, associatedUserID int64, db *pg.DB) (*models.Thread, *e.ErrorInfo) {
	var thread models.Thread
	var err error

	if threadID != 0 {
		err = db.Model(&thread).
			Where("thread_id = ?", threadID).
			Where("chat_id = ?", chatID).
			Select()
	} else if associatedUserID != 0 {
		err = db.Model(&thread).
			Where("chat_id = ?", chatID).
			Where("associated_user_id = ?", associatedUserID).
			Select()
	} else {
		return nil, e.NewError("either chatID or associatedUserID must be set", "Missing parameter").WithSeverity(e.Notice)
	}

	if err != nil {
		return nil, e.FromError(err, "Failed to fetch thread").WithSeverity(e.Notice).WithData(map[string]any{
			"threadID": threadID,
			"chatID": chatID,
			"associatedUserID": associatedUserID,
		})
	}

	return &thread, e.Nil()
}

func createThread(c tele.Context, targetChat *models.Chat, args *handlers.Arg) (*handlers.Arg, *e.ErrorInfo) {
	iconIDs := []string{
		"5386395194029515402",
		"5355127101970194557",
		"5350658016700013471",
		"5350367161514732241",
		"5309744892677727325",
		"5350751634102166060",
		"5309958691854754293",
		"5235912661102773458",
		"5413625003218313783",
		"5357504778685392027",
		"5310094636159607472",
	}

	t, err := c.Bot().CreateTopic(
		&tele.Chat{
			ID: targetChat.TgID,
		},
		&tele.Topic{
			Name: fmt.Sprintf("@%s", c.Sender().Username),
			IconCustomEmojiID: iconIDs[rand.Intn(len(iconIDs))],
		},
	)

	if err != nil {
		return args, e.FromError(err, "Failed to create topic").WithSeverity(e.Notice).WithData(map[string]any{
			"chatID": targetChat.TgID,
			"user": (*args)["user"],
		})
	}

	thread := &models.Thread{
		ThreadID: t.ThreadID,
		ChatID: targetChat.TgID,
		AssociatedUserID: (*args)["user"].(*models.User).TgID,
	}

	_, err = database.GetDB().Model(thread).Insert()
	if err != nil {
		return args, e.FromError(err, "Failed to insert thread").WithSeverity(e.Notice).WithData(map[string]any{
			"thread": thread,
			"chatID": targetChat.TgID,
			"user": (*args)["user"],
		})
	}

	(*args)["thread"] = thread

	return args, e.Nil()
}

func getOrCrateThreadFunc(c tele.Context, args *handlers.Arg) (*handlers.Arg, *e.ErrorInfo) {
	db := database.GetDB()

	if (*args)["sender_is_owner"].(bool) {
		thread, errInfo := getThread(c.Message().ThreadID, c.Chat().ID, 0, db)
		
		if errInfo.IsNotNil() {
			return args, errInfo.PushStack()
		}
		
		(*args)["thread"] = thread
		
		return args, e.Nil()
	}

	var targetChat models.Chat
	err := database.GetDB().Model(&targetChat).
		Where("chat_owner_id = ?", (*args)["target"].(*models.User).TgID).
		Select()
	if err != nil {
		return args, e.FromError(err, "Failed to select target chat").WithSeverity(e.Notice).WithData(map[string]any{
			"target": (*args)["target"],
		})
	}

	thread, errInfo := getThread(0, targetChat.TgID, (*args)["user"].(*models.User).TgID, db)

	if errInfo.IsNil() {
		(*args)["thread"] = thread
		return args, e.Nil()
	}

	if errInfo.Unwrap() != pg.ErrNoRows {
		return args, e.FromError(errInfo.Unwrap(), "Failed to select thread").WithSeverity(e.Notice).WithData(map[string]any{
			"threadID": c.Message().ThreadID,
			"chatID": c.Chat().ID,
			"associatedUserID": (*args)["user"].(*models.User).TgID,
			"user": (*args)["user"],
		})
	}

	args, errInfo = createThread(c, &targetChat, args)
	if errInfo.IsNotNil() {
		return args, errInfo.PushStack()
	}

	return args, e.Nil()
}

var (
	
)

var (
	GetSender = handlers.InitChainHandler(getSenderFunc)
	GetTarget = handlers.InitChainHandler(getTargetFunc)

	GetSenderAndTargetUser = handlers.InitChainHandler(
		getSenderAndTargetUserFunc,
		GetSender,
		GetTarget,
	)

	GetOrCrateThread = handlers.InitChainHandler(
		getOrCrateThreadFunc,
		GetSenderAndTargetUser,
	)
)