package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/volte6/mud/buffs"
	"github.com/volte6/mud/items"
	"github.com/volte6/mud/mobs"
	"github.com/volte6/mud/rooms"
	"github.com/volte6/mud/users"
	"github.com/volte6/mud/util"
)

func Give(rest string, userId int, cmdQueue util.CommandQueue) (util.MessageQueue, error) {

	response := NewUserCommandResponse(userId)

	// Load user details
	user := users.GetByUserId(userId)
	if user == nil { // Something went wrong. User not found.
		return response, fmt.Errorf("user %d not found", userId)
	}

	// Load current room details
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return response, fmt.Errorf(`room %d not found`, user.Character.RoomId)
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 2 {
		response.SendUserMessage(userId, "Give what? To whom?", true)
		response.Handled = true
		return response, nil
	}

	var giveItem items.Item = items.Item{}
	var giveGoldAmount int = 0

	if strings.ToLower(args[1]) == "gold" {

		g, _ := strconv.ParseInt(args[0], 10, 32)
		giveGoldAmount = int(g)

		if giveGoldAmount < 0 {
			response.SendUserMessage(userId, "You can't give a negative amount of gold.", true)
			response.Handled = true
			return response, nil
		}

		args = args[2:]

		if giveGoldAmount > user.Character.Gold {
			response.SendUserMessage(userId, "You don't have that much gold to give.", true)
			response.Handled = true
			return response, nil
		}

		if len(args) < 1 {
			response.SendUserMessage(userId, "Give it to whom?", true)
			response.Handled = true
			return response, nil
		}

	} else {

		var found bool = false

		// Check whether the user has an item in their inventory that matches
		giveItem, found = user.Character.FindInBackpack(args[0])

		if !found {
			response.SendUserMessage(userId, fmt.Sprintf("You don't have a %s to give.", args[0]), true)
			response.Handled = true
			return response, nil
		}

		args = args[1:]
	}

	playerId, mobId := room.FindByName(args[len(args)-1])

	if playerId > 0 {

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		targetUser := users.GetByUserId(playerId)

		// Swap the item location
		if giveItem.ItemId > 0 {
			targetUser.Character.StoreItem(giveItem)
			user.Character.RemoveItem(giveItem)

			iSpec := giveItem.GetSpec()
			if iSpec.QuestToken != `` {
				cmdQueue.QueueQuest(targetUser.UserId, iSpec.QuestToken)
			}

			response.SendUserMessage(userId,
				fmt.Sprintf(`You give the <ansi fg="item">%s</ansi> to <ansi fg="username">%s</ansi>.`, giveItem.Name(), targetUser.Character.Name),
				true)
			response.SendUserMessage(targetUser.UserId,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> gives you their <ansi fg="item">%s</ansi>.`, user.Character.Name, giveItem.Name()),
				true)
			response.SendRoomMessage(user.Character.RoomId,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> gives <ansi fg="username">%s</ansi> a <ansi fg="itemname">%s</ansi>.`, user.Character.Name, targetUser.Character.Name, giveItem.NameSimple()),
				true,
				user.UserId,
				targetUser.UserId)

		} else if giveGoldAmount > 0 {

			if targetUser.UserId == user.UserId {

				response.SendUserMessage(userId,
					fmt.Sprintf(`You count out <ansi fg="gold">%d gold</ansi> and put it back in your pocket.`, giveGoldAmount),
					true)
				response.SendRoomMessage(user.Character.RoomId,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> counts out some <ansi fg="gold">gold</ansi> and put it back in their pocket.`, user.Character.Name),
					true,
					user.UserId)

			} else {
				targetUser.Character.Gold += giveGoldAmount
				user.Character.Gold -= giveGoldAmount

				response.SendUserMessage(userId,
					fmt.Sprintf(`You give <ansi fg="gold">%d gold</ansi> to <ansi fg="username">%s</ansi>.`, giveGoldAmount, targetUser.Character.Name),
					true)
				response.SendUserMessage(targetUser.UserId,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> gives you <ansi fg="gold">%d gold</ansi>.`, user.Character.Name, giveGoldAmount),
					true)
				response.SendRoomMessage(user.Character.RoomId,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> gives <ansi fg="username">%s</ansi> some <ansi fg="gold">gold</ansi>.`, user.Character.Name, targetUser.Character.Name),
					true,
					user.UserId,
					targetUser.UserId)
			}
		} else {
			response.SendUserMessage(userId, "Something went wrong.", true)
		}

		response.Handled = true
		return response, nil

	}

	//
	// Look for an NPC
	//
	if mobId > 0 {

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		m := mobs.GetInstance(mobId)

		if m != nil {

			// Swap the item location
			if giveItem.ItemId > 0 || giveGoldAmount > 0 {

				if giveGoldAmount > 0 {
					m.Character.Gold += giveGoldAmount
					user.Character.Gold -= giveGoldAmount

					response.SendUserMessage(userId,
						fmt.Sprintf(`You give <ansi fg="gold">%d gold</ansi> to <ansi fg="username">%s</ansi>.`, giveGoldAmount, m.Character.Name),
						true)
					response.SendRoomMessage(room.RoomId,
						fmt.Sprintf(`<ansi fg="username">%s</ansi> gave some gold to <ansi fg="mobname">%s</ansi>.`, user.Character.Name, m.Character.Name),
						true)
				} else {

					m.Character.StoreItem(giveItem)
					user.Character.RemoveItem(giveItem)

					response.SendUserMessage(userId,
						fmt.Sprintf(`You give the <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, giveItem.Name(), m.Character.Name),
						true)
					response.SendRoomMessage(room.RoomId,
						fmt.Sprintf(`<ansi fg="username">%s</ansi> gave their <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, user.Character.Name, giveItem.Name(), m.Character.Name),
						true)

				}

				if len(m.ItemTrades) > 0 {

					for idx, trade := range m.ItemTrades {
						for _, itemId := range trade.AcceptedItemIds {

							if itemId == giveItem.ItemId {

								if _, ok := trade.GivenItems[userId]; !ok {
									trade.GivenItems[userId] = []int{}
								}

								alreadyGiven := false
								for _, givenItemId := range trade.GivenItems[userId] {
									if givenItemId == giveItem.ItemId {
										alreadyGiven = true
										break
									}
								}

								if !alreadyGiven {
									trade.GivenItems[userId] = append(trade.GivenItems[userId], giveItem.ItemId)
								}

								m.Character.RemoveItem(giveItem)
							}
						}

						if giveGoldAmount > 0 {
							if _, ok := trade.GivenGold[userId]; !ok {
								trade.GivenGold[userId] = 0
							}
							trade.GivenGold[userId] += giveGoldAmount
						}

						// Check whether all accepted items have been given
						goldGiven := trade.GivenGold[userId]

						if goldGiven >= trade.AcceptedGold && len(trade.GivenItems[userId]) == len(trade.AcceptedItemIds) {
							// If equal in length, then all items satisfied.
							for _, prizeItemId := range trade.PrizeItemIds {
								prizeItem := items.New(prizeItemId)
								if prizeItem.ItemId > 0 {
									m.Character.StoreItem(prizeItem)
									cmdQueue.QueueCommand(0, m.InstanceId, fmt.Sprintf(`give !%d to @%d`, prizeItem.ItemId, user.UserId))
								}
							}

							for _, prizeBuffId := range trade.PrizeBuffIds {
								cmdQueue.QueueBuff(user.UserId, 0, prizeBuffId)
							}

							if trade.PrizeRoomId > 0 {

								response.SendUserMessage(userId,
									`You are whisked away to a new location.`,
									true)
								response.SendRoomMessage(room.RoomId,
									fmt.Sprintf(`<ansi fg="username">%s</ansi> is whisked away to a new location.`, user.Character.Name),
									true)

								rooms.MoveToRoom(user.UserId, trade.PrizeRoomId)
							}

							for _, prizeQuestId := range trade.PrizeQuestIds {
								cmdQueue.QueueQuest(user.UserId, prizeQuestId)
							}

							for _, prizeCmd := range trade.PrizeCommands {
								cmdQueue.QueueCommand(0, m.InstanceId, prizeCmd)
							}

							if trade.PrizeGold > 0 {
								m.Character.Gold += trade.PrizeGold
								cmdQueue.QueueCommand(0, m.InstanceId, fmt.Sprintf(`give %d gold to @%d`, trade.PrizeGold, user.UserId))
							}

							delete(trade.GivenItems, userId)
							delete(trade.GivenGold, userId)
						}
						m.ItemTrades[idx] = trade
					}
				} else {
					cmdQueue.QueueCommand(0, mobId, fmt.Sprintf(`emote considers the <ansi fg="itemname">%s</ansi> for a moment.`, giveItem.Name()))
					cmdQueue.QueueCommand(0, mobId, fmt.Sprintf(`gearup !%d`, giveItem.ItemId))
				}

			} else {
				response.SendUserMessage(userId, "Something went wrong.", true)
			}

		}

		response.Handled = true
		return response, nil
	}

	response.SendUserMessage(userId, "Who???", true)

	response.Handled = true
	return response, nil
}
