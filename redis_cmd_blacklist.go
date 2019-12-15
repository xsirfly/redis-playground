package main

// Blacklist a list of Redis commands which we do not allow to execute.
// type Blacklist []string
type Blacklist map[string]bool

var redisCmdBlacklist = Blacklist{"CLUSTER": true, "CONFIG": true, "DEBUG": true, "SLAVEOF": true, "RESTORE": true, "MIGRATE": true}
