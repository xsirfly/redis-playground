package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/kballard/go-shellquote"
)

const (
	redisConnectTimeout = 10 * time.Second
	redisReadTimeout    = 10 * time.Second
	redisWriteTimeout   = 10 * time.Second
)

// ClientRedisCommand represents a tuple containing client and its Redis command.
type ClientRedisCommand struct {
	client  *Client
	command []byte
}

// Hub maintains the set of active clients and proxy commands to Redis.
type Hub struct {
	// Registered clients.
	client *Client

	// Inbound messages from the clients.
	broadcast chan ClientRedisCommand

	// Connection to Redis.
	c redis.Conn
}

func newHub() *Hub {
	return &Hub{
		broadcast: make(chan ClientRedisCommand),
	}
}

func (h *Hub) parseRequest(request []byte) (string, []interface{}) {
	// Extract command name.
	var args []interface{}

	command := string(request[:])
	// z := strings.SplitN(command, " ", 2)
	z, _ := shellquote.Split(command)
	commandName := z[0]
	if len(z) > 1 {
		commandArgs := z[1:]
		// Convert request arguments from string array to interface{} array.
		args = make([]interface{}, len(commandArgs))
		for i, v := range commandArgs {
			args[i] = v
		}
	}

	return commandName, args
}

func (h *Hub) parseReply(reply interface{}, err error) string {
	if err != nil {
		return fmt.Sprintf("%s", err.Error())
	}

	switch reply.(type) {
	case int64:
		i, _ := redis.Int64(reply, err)
		return fmt.Sprintf("%d", i)
	case string:
		s, _ := redis.String(reply, err)
		return s
	case []byte:
		b, _ := redis.Bytes(reply, err)
		return fmt.Sprintf("%s", b)
	case []interface{}:
		s := ""
		values, _ := redis.Values(reply, err)
		for i := 0; i < len(values); i++ {
			s += h.parseReply(values[i], nil) + "\n"
		}
		return s
	case nil:
		return "(nil)"
	default:
		// We shouldn't reach this point
		panic("Unknown reply type.")
	}
}

func (h *Hub) executeCommand(command string, args []interface{}) string {
	reply, err := h.c.Do(command, args...)
	return h.parseReply(reply, err)
}

func (h *Hub) run(ip string) {
	var err error
	// Give Redis some time before trying to connect.
	time.Sleep(5 * time.Second)
	h.c, err = redis.DialTimeout("tcp", ip+":6379", redisConnectTimeout, redisReadTimeout, redisWriteTimeout)

	if err != nil {
		log.Printf("Could not establish connection to redis: %v", err)
		return
	}

	defer h.c.Close()

	for {
		select {
		case request := <-h.broadcast:
			message := request.command
			client := request.client
			var response string
			commandName, args := h.parseRequest(message)

			// Make sure command isn't black listed.
			if _, ok := redisCmdBlacklist[strings.ToUpper(commandName)]; ok {
				response = fmt.Sprintf("ERR Unknown or disabled command '%s'", commandName)
			} else {
				response = h.executeCommand(commandName, args)
			}

			select {
			case client.send <- []byte(response):
			default:
				close(client.send)
			}
		}
	}
}
