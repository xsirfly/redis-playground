package main

import (
	"sort"
	"strings"

	"github.com/gomodule/redigo/redis"
)

func b2s(bs []uint8) string {
	b := make([]byte, len(bs))
	for i, v := range bs {
		b[i] = byte(v)
	}
	return string(b)
}

// RedisCommandList extracts Redis commands list
func RedisCommandList(c redis.Conn) ([]string, error) {
	reply, err := redis.Values(c.Do("COMMAND"))
	if err != nil {
		return nil, err
	}

	commands := make([]string, len(reply))

	for i := 0; i < len(reply); i++ {
		command := reply[i].([]interface{})
		commandName := b2s(command[0].([]uint8))
		commands[i] = strings.ToUpper(commandName)
	}

	sort.Strings(commands)
	return commands, nil
}
