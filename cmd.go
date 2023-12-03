package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hdt3213/rdb/helper"
)

const help = `
This is a tool to parse Redis' RDB files
Options:
  -c command, including: json/memory/aof/bigkey/flamegraph/ssv
  -o output file path
  -n number of result, using in 
  -port listen port for flame graph web service
  -sep separator for flamegraph, rdb will separate key by it, default value is ":". 
		supporting multi separators: -sep sep1 -sep sep2 
  -regex using regex expression filter keys
  -no-expired filter expired keys
  -field including: [db]/key/expiration/size/type/encoding/value/values/hash/members/entries

Examples:
parameters between '[' and ']' is optional
1. convert rdb to json
  rdb -c json -o dump.json dump.rdb
2. generate memory report
  rdb -c memory -o memory.csv dump.rdb
3. convert to aof file
  rdb -c aof -o dump.aof dump.rdb
4. get largest keys
  rdb -c bigkey [-o dump.aof] [-n 10] dump.rdb
5. draw flamegraph
  rdb -c flamegraph [-port 16379] [-sep :] dump.rdb
6. convert rdb to [json/ssv] with fields
  rdb -c ssv -fields key,size,type -regex FULL -o dump.list dump.rdb
`

type separators []string

func (s *separators) String() string {
	return strings.Join(*s, " ")
}

func (s *separators) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	var cmd string
	var output string
	var n int
	var port int
	var seps separators
	var regexExpr string
	var noExpired bool
	var fields string
	flagSet.StringVar(&cmd, "c", "", "command for rdb: json")
	flagSet.StringVar(&output, "o", "", "output file path")
	flagSet.IntVar(&n, "n", 0, "")
	flagSet.IntVar(&port, "port", 0, "listen port for web")
	flagSet.Var(&seps, "sep", "separator for flame graph")
	flagSet.StringVar(&regexExpr, "regex", "", "regex expression")
	flagSet.BoolVar(&noExpired, "no-expired", false, "filter expired keys")
	flagSet.StringVar(&fields, "fields", "", "output fields")

	_ = flagSet.Parse(os.Args[1:]) // ExitOnError
	src := flagSet.Arg(0)

	if cmd == "" {
		println(help)
		return
	}
	if src == "" {
		println("src file is required")
		return
	}

	var options []interface{}
	if regexExpr != "" {
		options = append(options, helper.WithRegexOption(regexExpr))
	}
	if noExpired {
		options = append(options, helper.WithNoExpiredOption())
	}
	if fields != "" {
		options = append(options, helper.WithFieldOption(fields))
	}

	var err error
	switch cmd {
	case "json":
		err = helper.ToJsons(src, output, options...)
	case "ssv":
		err = helper.ToSSV(src, output, options...)
	case "aof":
		err = helper.ToAOF(src, output, options)
	case "memory":
		err = helper.MemoryProfile(src, output, options...)
	case "bigkey":
		if output == "" {
			err = helper.FindBiggestKeys(src, n, os.Stdout, options...)
		} else {
			var outputFile *os.File
			outputFile, err = os.Create(output)
			if err != nil {
				fmt.Printf("open output faild: %v", err)
			}
			defer func() {
				_ = outputFile.Close()
			}()
			err = helper.FindBiggestKeys(src, n, outputFile, options...)
		}
	case "flamegraph":
		_, err = helper.FlameGraph(src, port, seps, options...)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}
		<-make(chan struct{})
	default:
		println("unknown command")
		return
	}
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
}
