package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/mediocregopher/radix.v2/redis"
	"github.com/peterh/liner"
)

var (
	uri      = flag.String("u", "", "Server URI.")
	hostname = flag.String("h", "127.0.0.1", "Server hostname.")
	port     = flag.Int("p", 6379, "Server port.")
	dbn      = flag.Int("n", 0, "Database number.")
	auth     = flag.String("a", "", "Password to use when connecting to the server.")
	secure   = flag.Bool("tls", false, "Connect over SSL/TLS.")

	line   *liner.State
	client *redis.Client

	historyPath = path.Join(os.Getenv("HOME"), ".rediscli_history")
	prompt      = "> "
)

func main() {
	flag.Parse()

	startRedisClient()

	if flag.NArg() == 0 {
		repl()
	} else {
		processCommand(flag.Args())
	}
}

func repl() {
	line = liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	setCompletionHandler()
	loadHistory()
	defer saveHistory()

	reg, _ := regexp.Compile(`'.*?'|".*?"|\S+`)

	for {
		inputCommand, err := line.Prompt(prompt)
		if err != nil {
			switch err {
			case liner.ErrPromptAborted:
			default:
				fmt.Printf("%s\n", err.Error())
			}
			return
		}

		splitCommand := reg.FindAllString(inputCommand, -1)

		if len(splitCommand) == 0 {
			continue
		} else if s := strings.ToUpper(splitCommand[0]); s == "QUIT" || s == "EXIT" {
			break
		}

		appendHistory(splitCommand)

		processCommand(splitCommand)
	}
}

func processCommand(cmd []string) {
	sliceCommand := make([]interface{}, len(cmd))
	for i, v := range cmd {
		sliceCommand[i] = v
	}

	res := client.Cmd(cmd[0], sliceCommand[1:]...)
	printResultNew(res, "")
}

func printResultNew(res *redis.Resp, prefix string) {
	if res.IsType(redis.Nil) {
		fmt.Printf("(nil)\n")
		return
	}

	if res.Err != nil {
		fmt.Printf("(error) %s\n", res.Err.Error())
		return
	}

	if res.IsType(redis.Int) {
		i, _ := res.Int()
		fmt.Printf("(integer) %d\n", i)
	} else if res.IsType(redis.SimpleStr) {
		s, _ := res.Str()
		fmt.Printf("%s\n", s)
	} else if res.IsType(redis.BulkStr) {
		s, _ := res.Str()
		fmt.Printf("%q\n", s)
	} else {
		a, _ := res.Array()

		if len(a) == 0 {
			fmt.Println("(empty list or set)")
			return
		}

		i := len(a)
		idxLen := 0
		for i != 0 {
			idxLen = idxLen + 1
			i /= 10
		}

		_prefix := strings.Repeat(" ", idxLen+2)
		_prefixfmt := fmt.Sprintf("%%s%%%dd) ", idxLen)

		for i, v := range a {
			var p string

			if i == 0 {
				p = ""
			} else {
				p = prefix
			}
			fmt.Printf(_prefixfmt, p, i+1)

			printResultNew(v, _prefix)
		}
	}
}

func startRedisClient() {
	addr := fmt.Sprintf("%s:%d", *hostname, *port)

	var err error
	if *secure {
		conn, _ := tls.Dial("tcp", addr, &tls.Config{ServerName: *hostname})
		client, err = redis.NewClient(conn)
	} else {
		client, err = redis.Dial("tcp", addr)
	}
	if err != nil {
		panic("failed to start client")
	}

	if *auth != "" {
		if err = client.Cmd("AUTH", *auth).Err; err != nil {
			client.Close()
			os.Exit(1)
		}
	}
}

func setCompletionHandler() {
	line.SetCompleter(func(line string) (c []string) {
		for _, i := range autocompleteCommands {
			if strings.HasPrefix(i, strings.ToUpper(line)) {
				c = append(c, i)
			}
		}
		return
	})
}

func loadHistory() {
	if f, err := os.Open(historyPath); err == nil {
		line.ReadHistory(f)
		f.Close()
	}
}

func appendHistory(cmds []string) {
	cloneCmds := make([]string, len(cmds))
	copy(cloneCmds, cmds)

	line.AppendHistory(strings.Join(cloneCmds, " "))
}

func saveHistory() {
	if f, err := os.Create(historyPath); err != nil {
		fmt.Printf("Error writing history file: %s", err.Error())
	} else {
		line.WriteHistory(f)
		f.Close()
	}
}
