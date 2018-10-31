package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/go-redis/redis"
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

	redisCommand := redis.NewCmd(sliceCommand...)
	client.Process(redisCommand)
	res, err := redisCommand.Result()
	printResult(res, err, "")
}

func printResult(res interface{}, err error, prefix string) {
	if res == nil {
		fmt.Printf("(nil)\n")
		return
	}

	if err != nil {
		fmt.Printf("(error) %s\n", err.Error())
		return
	}

	switch res := res.(type) {
	case int64:
		fmt.Printf("(integer) %d\n", res)
	case string:
		fmt.Printf("%s\n", res)
	case []interface{}:
		if len(res) == 0 {
			fmt.Println("(empty list or set)")
			return
		}

		i := len(res)
		idxLen := 0
		for i != 0 {
			idxLen = idxLen + 1
			i /= 10
		}

		_prefix := strings.Repeat(" ", idxLen+2)
		_prefixfmt := fmt.Sprintf("%%s%%%dd) ", idxLen)

		for i, v := range res {
			var p string

			if i == 0 {
				p = ""
			} else {
				p = prefix
			}
			fmt.Printf(_prefixfmt, p, i+1)

			printResult(v, nil, _prefix)
		}
	default:
		fmt.Printf("Unknown reply type: %v\n", res)
	}
}

func startRedisClient() {
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", *hostname, *port),
		Password: *auth,
		DB:       *dbn,
	}

	if *secure {
		opts.TLSConfig = &tls.Config{ServerName: *hostname}
	}

	if *uri != "" {
		o, err := redis.ParseURL(*uri)
		if err != nil {
			panic(err)
		}
		opts = o
	}

	client = redis.NewClient(opts)
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
