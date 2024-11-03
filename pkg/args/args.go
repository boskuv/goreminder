package args

import (
	arg "github.com/alexflint/go-arg"
)

var args Arguments

type Arguments struct {
	ConfigPath string `default:"./config.yaml"`
}

func ParseArgs() *Arguments {
	arg.MustParse(&args)
	return &args
}
