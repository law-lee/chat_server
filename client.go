package main

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

// client represents a single chatting user.
type client struct {
	// socket is the web socket for this client.
	socket *websocket.Conn
	// send is a channel on which messages are sent.
	send chan *message
	// room is the room this client is chatting in.
	room *room
	// userData holds information about the user
	userData map[string]interface{}
}

func (c *client) read() {
	defer c.closeSocket()
	for {
		var msg *message
		err := c.socket.ReadJSON(&msg)
		if err != nil {
			return
		}
		msg.When = time.Now()
		msg.Name = c.userData["name"].(string)
		// assigned a value to AvatarURL
		//All we have done here is take the value from the userData field that represents what we
		//put into the cookie and assigned it to the appropriate field in message if the value was
		//present in the map
		//if avatarURL, ok := c.userData["avatar_url"]; ok {
		//	msg.AvatarURL = avatarURL.(string)
		//}
		if avatarUrl, ok := c.userData["avatar_url"]; ok {
			msg.AvatarURL = avatarUrl.(string)
		}
		c.room.forward <- msg
	}
}
func (c *client) write() {
	defer c.closeSocket()
	for msg := range c.send {
		err := c.socket.WriteJSON(msg)
		if err != nil {
			break
		}
	}
}

func (c *client) closeSocket() {
	if err := c.socket.Close(); err != nil {
		fmt.Printf("close socket err: %v", err)
	}
}
