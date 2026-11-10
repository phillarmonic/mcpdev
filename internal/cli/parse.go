package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/model"
)

// Invocation is parsed argv before config merge.
type Invocation struct {
	TraceFormat     model.Format
	ArgsJSON        string
	ArgsFile        string
	CWD             string
	Protocol        string
	Template        string
	Section         string
	SavePath        string
	Out             string
	From            string
	To              string
	FromURL         string
	ToURL           string
	FromProfile     string
	ToProfile       string
	ToPath          string
	FailOn          string
	URL             string
	Format          model.Format
	Command         string
	Profile         string
	JUnit           string
	Record          string
	Replay          string
	Readiness       string
	OnSuccess       string
	Artifact        string
	Staging         string
	Baseline        string
	Ignore          []string
	CommandArgv     []string
	Paths           []string
	Names           []string
	FromCommand     []string
	RebuildArgv     []string
	WatchRoots      []string
	Include         []string
	Exclude         []string
	Smoke           []string
	Headers         []string
	Renames         []string
	Env             []string
	ToCommand       []string
	ArgPairs        []string
	MaxBytes        int
	Timeout         time.Duration
	Debounce        time.Duration
	Jobs            int
	ConnectTimeout  time.Duration
	ConnectSet      bool
	TimeoutSet      bool
	FormatSet       bool
	ProtocolSet     bool
	ExitOnChangeSet bool
	DebounceSet     bool
	KeepLastGoodSet bool
	AllowIncomplete bool
	SkipDiscover    bool
	ExitOnChange    bool
	Trace           bool
	Verbose         bool
	FailOnToolError bool
	Expand          bool
	FailFast        bool
	KeepLastGood    bool
	Insecure        bool
	AllowHTTPSse    bool
	Help            bool
}

func parseArgs(args []string) (Invocation, error) {
	var in Invocation
	in.ExitOnChange = true
	i := 0
	if len(args) > 0 {
		i = 1
	}
	var positionals []string
	for i < len(args) {
		a := args[i]
		if a == "--" {
			in.CommandArgv = append(in.CommandArgv, args[i+1:]...)
			break
		}
		switch {
		case a == "-h" || a == "--help":
			in.Help = true
			i++
		case a == "--insecure":
			in.Insecure = true
			i++
		case a == "--no-discover":
			in.SkipDiscover = true
			i++
		case a == "--allow-http-sse":
			in.AllowHTTPSse = true
			i++
		case a == "--allow-incomplete":
			in.AllowIncomplete = true
			i++
		case a == "--exit-on-change":
			in.ExitOnChange = true
			in.ExitOnChangeSet = true
			i++
		case a == "--no-exit-on-change":
			in.ExitOnChange = false
			in.ExitOnChangeSet = true
			i++
		case a == "--fail-fast":
			in.FailFast = true
			i++
		case a == "--keep-last-good":
			in.KeepLastGood = true
			in.KeepLastGoodSet = true
			i++
		case a == "--no-keep-last-good":
			in.KeepLastGood = false
			in.KeepLastGoodSet = true
			i++
		case a == "--expand":
			in.Expand = true
			i++
		case a == "--fail-on-tool-error":
			in.FailOnToolError = true
			i++
		case a == "--trace":
			in.Trace = true
			i++
		case a == "-v" || a == "--verbose":
			in.Verbose = true
			i++
		case a == "-c" || a == "--command":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.CommandArgv = append(in.CommandArgv, v)
			i = n
		case a == "--url":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.URL = v
			i = n
		case a == "-p" || a == "--profile":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Profile = v
			i = n
		case a == "--header":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Headers = append(in.Headers, v)
			i = n
		case a == "--env":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Env = append(in.Env, v)
			i = n
		case a == "--cwd":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.CWD = v
			i = n
		case a == "--timeout":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			d, err := time.ParseDuration(v)
			if err != nil {
				return Invocation{}, errclass.Usage{Err: err, Msg: "--timeout"}
			}
			in.Timeout = d
			in.TimeoutSet = true
			i = n
		case a == "--connect-timeout":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			d, err := time.ParseDuration(v)
			if err != nil {
				return Invocation{}, errclass.Usage{Err: err, Msg: "--connect-timeout"}
			}
			in.ConnectTimeout = d
			in.ConnectSet = true
			i = n
		case a == "--protocol":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Protocol = v
			in.ProtocolSet = true
			i = n
		case a == "--trace-format":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			switch v {
			case "human":
				in.TraceFormat = model.FormatHuman
			case "json":
				in.TraceFormat = model.FormatJSON
			default:
				return Invocation{}, errclass.Usage{Msg: "unknown --trace-format " + v}
			}
			i = n
		case a == "-o" || a == "--format":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			switch v {
			case "human":
				in.Format = model.FormatHuman
			case "json":
				in.Format = model.FormatJSON
			default:
				return Invocation{}, errclass.Usage{Msg: "unknown --format " + v}
			}
			in.FormatSet = true
			i = n
		case a == "--name":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Names = append(in.Names, v)
			i = n
		case a == "--args":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.ArgsJSON = v
			i = n
		case a == "--args-file":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.ArgsFile = v
			i = n
		case a == "--arg":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.ArgPairs = append(in.ArgPairs, v)
			i = n
		case a == "--out":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Out = v
			i = n
		case a == "--from":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.From = v
			i = n
		case a == "--to":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.To = v
			i = n
		case a == "--from-url":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.FromURL = v
			i = n
		case a == "--to-url":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.ToURL = v
			i = n
		case a == "--from-profile":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.FromProfile = v
			i = n
		case a == "--to-profile":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.ToProfile = v
			i = n
		case a == "--from-command":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.FromCommand = append(in.FromCommand, v)
			i = n
		case a == "--to-command":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.ToCommand = append(in.ToCommand, v)
			i = n
		case a == "--fail-on":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.FailOn = v
			i = n
		case a == "--ignore":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Ignore = append(in.Ignore, v)
			i = n
		case a == "--rename":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Renames = append(in.Renames, v)
			i = n
		case a == "--save":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.SavePath = v
			i = n
		case a == "--template":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Template = v
			i = n
		case a == "--record":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Record = v
			i = n
		case a == "--replay":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Replay = v
			i = n
		case a == "--junit":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.JUnit = v
			i = n
		case a == "--jobs":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			nb, convErr := strconv.Atoi(v)
			if convErr != nil || nb < 1 {
				return Invocation{}, errclass.Usage{Msg: "--jobs must be a positive integer"}
			}
			in.Jobs = nb
			i = n
		case a == "--rebuild":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.RebuildArgv = append(in.RebuildArgv, v)
			i = n
		case a == "--root":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.WatchRoots = append(in.WatchRoots, v)
			i = n
		case a == "--include":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Include = append(in.Include, v)
			i = n
		case a == "--exclude":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Exclude = append(in.Exclude, v)
			i = n
		case a == "--smoke":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Smoke = append(in.Smoke, v)
			i = n
		case a == "--baseline":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Baseline = v
			i = n
		case a == "--staging":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Staging = v
			i = n
		case a == "--artifact":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Artifact = v
			i = n
		case a == "--on-success":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.OnSuccess = v
			i = n
		case a == "--readiness":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			in.Readiness = v
			i = n
		case a == "--debounce":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			d, err := time.ParseDuration(v)
			if err != nil {
				return Invocation{}, errclass.Usage{Err: err, Msg: "--debounce"}
			}
			in.Debounce = d
			in.DebounceSet = true
			i = n
		case a == "--max-bytes":
			v, n, err := takeValue(args, i, a)
			if err != nil {
				return Invocation{}, err
			}
			nb, convErr := strconv.Atoi(v)
			if convErr != nil || nb < 1 {
				return Invocation{}, errclass.Usage{Msg: "--max-bytes must be a positive integer"}
			}
			in.MaxBytes = nb
			i = n
		case strings.HasPrefix(a, "-"):
			return Invocation{}, errclass.Usage{Msg: "unknown flag " + a}
		default:
			positionals = append(positionals, a)
			i++
		}
	}
	if len(positionals) > 0 {
		in.Command = positionals[0]
	}
	switch in.Command {
	case "test":
		in.Paths = append([]string{}, positionals[1:]...)
		if len(in.Paths) > 0 {
			in.Section = in.Paths[0]
		}
	case "diff":
		if len(positionals) > 1 {
			in.Section = positionals[1]
		}
		if len(positionals) > 2 {
			in.ToPath = positionals[2]
		}
		if len(positionals) > 3 {
			return Invocation{}, errclass.Usage{Msg: "unexpected argument " + positionals[3]}
		}
	default:
		if len(positionals) > 1 {
			in.Section = positionals[1]
		}
		if len(positionals) > 2 {
			return Invocation{}, errclass.Usage{Msg: "unexpected argument " + positionals[2]}
		}
	}
	return in, nil
}

func takeValue(args []string, i int, flag string) (string, int, error) {
	if i+1 >= len(args) {
		return "", 0, errclass.Usage{Msg: flag + " requires a value"}
	}
	return args[i+1], i + 2, nil
}

func parseSection(s string) ([]model.PrimitiveKind, error) {
	if s == "" || s == "all" {
		return nil, nil
	}
	switch model.PrimitiveKind(s) {
	case model.KindTools, model.KindPrompts, model.KindResources, model.KindTemplates:
		return []model.PrimitiveKind{model.PrimitiveKind(s)}, nil
	default:
		return nil, errclass.Usage{Msg: fmt.Sprintf("unknown inspect section %q", s)}
	}
}
