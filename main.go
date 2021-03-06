package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PaesslerAG/gval"
	"github.com/adrg/xdg"
	"github.com/gabriel-vasile/mimetype"
	"github.com/google/shlex"
	"gopkg.in/ini.v1"
)

type Config struct {
	Browser         []string
	ProgramLauncher []string
	FilterPath      string
	FilterShell     []string
	ClipboardCmd    []string
	Detach          bool
	TypeHandlers    []Handler
}

type Handler struct {
	Name            string
	Program         []string
	Extensions      []string
	Filters         []string
	MimeTypes       []string
	Protocols       []string
	UrlRegexs       []string
	MatchExpression string
}

const (
	LOG_NONE int = iota
	LOG_ERROR
	LOG_WARNING
	LOG_DEBUG
)

var (
	AppName       string
	debug         *bool
	VersionNumber string
	loglevel      int
)

func log(level int, format string, args ...interface{}) {
	if level <= loglevel {
		fmt.Printf(format, args...)
	}
}

func get_mime_type(ressource *URL) (string, error) {

	var (
		mime string
		err  error
	)

	switch ressource.Scheme {
	case "http":
		fallthrough
	case "https":
		resp, err := http.Head(ressource.String())
		if err == nil {
			resp := resp.Header["Content-Type"]
			if resp != nil {
				mime = resp[0]
			}
		}
	case "file":
		fallthrough
	case "":
		// check if path is a directory, the library does not handle them
		// TODO Maybe I could ask if they want to support directories as well but I doubt it
		info, err := os.Stat(ressource.Path)
		if !os.IsNotExist(err) && info.IsDir() {
			return "inode/directory", nil
		}

		mtype, err_ := mimetype.DetectFile(ressource.Path)
		if err == nil {
			mime = mtype.String()
		} else {
			log(LOG_ERROR, "Error getting mime type: %v\n", err_)
		}
		// FIXME Yikes, I should look into variable shadowing rules but I think err was being create
		// as a new variable here and therefor th function did not return the error
		err = err_
	}
	return mime, err
}

func run_filter(filter string, url *URL, config *Config, handler Handler) (bool, string) {
	var executable string
	if config.FilterPath != "" {
		executable = filepath.Join(config.FilterPath, filter)
		_, err := os.Stat(executable)
		if err != nil {
			if handler.Name != filter {
				log(LOG_WARNING, "Could not run the filter \"%s\": %s\n", filter, err)
			}
			return false, ""
		}
	} else {
		executable = filter
	}
	cmdline := config.FilterShell
	cmdline = append(cmdline, executable)
	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	env := os.Environ()
	env = append(env, fmt.Sprintf("url=%s", url.String())) //XXX this could be slitghly different from raw_url
	env = append(env, fmt.Sprintf("protocol=%s", url.Scheme))
	env = append(env, fmt.Sprintf("user=%s", url.User.Username()))
	env = append(env, fmt.Sprintf("host=%s", url.Host))
	env = append(env, fmt.Sprintf("path=%s", url.Path))
	env = append(env, fmt.Sprintf("section=%s", handler.Name))
	cmd_stdin, err := cmd.StdinPipe()
	if err != nil {
		log(LOG_ERROR, "Could not get a pipe to the filters stdin: %v\n", err)
	}
	pwd, available := url.User.Password()
	if available && err == nil {
		cmd_stdin.Write([]byte(pwd))
	}
	cmd_stdin.Close()
	cmd_stdout, err := cmd.StdoutPipe()
	if err != nil {
		log(LOG_ERROR, "Could not get a pipe to the filters stdout: %v\n", err)
	}
	cmd.Env = env
	log(LOG_DEBUG, "Running %#v\n", cmdline)
	err = cmd.Run()
	if err != nil && cmd.ProcessState.ExitCode() != 1 {
		fmt.Fprintf(os.Stderr, "Error running the filter \"%s\": %s\n", filter, err)
	} else if cmd.ProcessState.ExitCode() == 0 {
		new_url, err := io.ReadAll(cmd_stdout)
		if err != nil {
			log(LOG_ERROR, "Error reading stdout from the filter (%s) output: %v\n", filter, err)
		}
		return true, string(new_url)
	} else if cmd.ProcessState.ExitCode() == 1 {
		log(LOG_WARNING, "Filter (%s) returned 1\n", filter)
		return false, ""
	}
	return false, ""
}

func handle_uri(raw_url string, config *Config) {

	var (
		extension string
		mime_type string
	)
	url, err := Parse(raw_url)
	if err != nil {
		log(LOG_ERROR, "Error in parsing the url %v\n", err)
		url = &URL{
			raw_url: raw_url,
		}
		extension = ""
		mime_type = ""
	} else {
		parts := strings.Split(url.Path, ".")
		extension = parts[len(parts)-1]
		var err error //TODO Still havent looked into shadowing rules
		mime_type, err = get_mime_type(url)
		if err != nil {
			log(LOG_WARNING, "Could not get mime type for %s: %v\n", url.String(), err)
		}
	}

	runner := config.Browser

	for _, handler := range config.TypeHandlers {
		params := map[string]bool{
			"protocol":  false,
			"mime_type": false,
			"extension": false,
			"url_regex": false,
			"filter":    false,
		}
		name := handler.Name
		log(LOG_DEBUG, "Checking matchs for %s\n", name)
		for _, p := range handler.Protocols {
			if p == url.Scheme {
				log(LOG_DEBUG, "Matched with the protocol for %#v\n", p)
				params["protocol"] = true
			}
		}
		for _, mime := range handler.MimeTypes {
			matched, err := regexp.MatchString(mime, mime_type)
			if err != nil {
				log(LOG_ERROR, "[mime]: %s is not a valide regex, ignored...\n", mime)
				continue
			}
			if matched {
				log(LOG_DEBUG, "Matched with the mime-type for %#v\n", mime)
				params["mime_type"] = true
			}
		}

		for _, ext := range handler.Extensions {
			if ext == extension {
				log(LOG_DEBUG, "Matched with the extension for %#v\n", ext)
				params["extension"] = true
			}
		}
		for _, reg := range handler.UrlRegexs {
			matched, err := regexp.MatchString(reg, raw_url)
			if err != nil {
				log(LOG_ERROR, "[url_regex]: %s is not a valide regex, ignored...\n", reg)
				continue
			} else if matched {
				log(LOG_DEBUG, "Matched with a regex for %#v\n", reg)
				params["url_regex"] = true
			}
		}

		filter_handler := func(filter string) {
			//TODO: support arguments for filters?
			match, new_url := run_filter(filter, url, config, handler)
			if new_url != "" {
				//TODO: do something to replace the current url
				log(LOG_DEBUG, "(%s):A filter gave us a new url: %s\n", filter, new_url)
			}
			if match {
				log(LOG_DEBUG, "Matched because of a filter: %s\n", filter)
				params["filter"] = true
			}
		}
		if len(handler.Filters) == 0 {
			filter_handler(name)
		}
		for _, filter := range handler.Filters {
			filter_handler(filter)
		}
		matched, err := gval.Evaluate(handler.MatchExpression, params)
		if err != nil {
			log(LOG_ERROR, "Error in evaluating the MatchExpression: %v\n", err)
		} else if matched == true {
			log(LOG_DEBUG, "Matched the url for section %s\n", name)
			// matched is an interface so I cannot just evaluate it in the condition
			runner = handler.Program
			break
		}
	}

	var cmdline []string
	if len(config.ProgramLauncher) > 0 {
		cmdline = append(cmdline, config.ProgramLauncher...)
	}

	cmdline = append(cmdline, runner...)
	if url.Scheme == "file" {
		cmdline = append(cmdline, url.Path)
	} else {
		// TODO handle special character getting mangled in encoding/decoding
		cmdline = append(cmdline, url.String())
	}
	cmd := exec.Command(cmdline[0], cmdline[1:]...)

	log(LOG_DEBUG, "Handling the url with: %#v\n", cmdline)

	if config.Detach {
		err = cmd.Start()
	} else {
		err = cmd.Run()
	}
	if err != nil {
		log(LOG_ERROR, "Error running the command: %v\n", err)
		return
	}
}

func expandTilde(s string) string {
	if strings.HasPrefix(s, "~/") {
		dirname, _ := os.UserHomeDir()
		s = filepath.Join(dirname, s[2:])
	}
	return s
}

func main() {
	flag.Usage = func() {
		exec_name := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage %s:\n", exec_name)
		fmt.Fprintf(os.Stderr, "  %s [OPTIONS] URIs...\n", exec_name)
		fmt.Fprintf(os.Stderr, "  some_command | %s [OPTIONS]\n", exec_name)
		fmt.Fprintf(os.Stderr, "  %s [OPTIONS] (will read from the clipboard)\n", exec_name)
		fmt.Fprintf(os.Stderr, "\nOPTIONS\n")
		fmt.Fprintf(os.Stderr, "    -loglevel {none|warning|error|debug} Log level   \n")
		fmt.Fprintf(os.Stderr, "    -help                                Print the usage and exit\n")
		fmt.Fprintf(os.Stderr, "    -version                             Print the version and exit\n")
		fmt.Fprintf(os.Stderr, "\n")
	}

	config := &Config{
		Browser: []string{"xdg-open"},
	}

	loglevelFlag := flag.String("loglevel", "none", "Set the loglevel")
	version := flag.Bool("version", false, "Print the version")
	help := flag.Bool("help", false, "Print help")
	flag.Parse()

	if len(AppName) == 0 {
		AppName = filepath.Base(os.Args[0])
	}

	if *help {
		flag.Usage()
		return
	}

	switch *loglevelFlag {
	case "warning":
		loglevel = LOG_WARNING
	case "error":
		loglevel = LOG_ERROR
	case "debug":
		loglevel = LOG_DEBUG
	case "none":
		loglevel = LOG_NONE
	default:
		flag.Usage()
		fmt.Fprintf(os.Stderr, "%s is not a valid loglevel\n", *loglevelFlag)
		return
	}

	if *version {
		fmt.Fprintf(os.Stderr, "%s: version %s\n", AppName, VersionNumber)
		return
	}

	configFile, err := xdg.SearchConfigFile(filepath.Join(AppName, "config.ini"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not load config file: %v\n", err)
		return
	}

	conf, err := ini.ShadowLoad(configFile)
	conf.ValueMapper = func(s string) string {
		s = os.ExpandEnv(s)
		return expandTilde(s)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error oppening config file: %v", err)
		return
	}

	for _, section := range conf.Sections() {
		if section.Name() == "DEFAULT" {
			prog_cmd_line := section.Key("browser").String()
			config.Browser, err = shlex.Split(prog_cmd_line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Cannot understand %s as a shell command (from the %s section)\n",
					prog_cmd_line, section.Name())
				return
			}

			launcher_cmd_line := section.Key("program_launcher").String()
			config.ProgramLauncher, err = shlex.Split(launcher_cmd_line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Cannot understand %s as a shell command (from the %s section)\n",
					prog_cmd_line, section.Name())
				return
			}

			config.FilterPath = section.Key("filter_path").String()
			if _, err := os.Stat(config.FilterPath); config.FilterPath != "" && os.IsNotExist(err) {
				fmt.Printf("Error with the filter_path: %s\n", err)
				return
			}
			filter_shell_cmd_line := section.Key("filter_shell").String()
			config.FilterShell, err = shlex.Split(filter_shell_cmd_line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Cannot understand %s as a shell command (from the %s section)\n",
					prog_cmd_line, section.Name())
				return
			}

			clipboard_cmd_line := section.Key("clipboard_cmd").String()
			config.ClipboardCmd, err = shlex.Split(clipboard_cmd_line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Cannot understand %s as a shell command (from the %s section)\n",
					clipboard_cmd_line, section.Name())
				return
			}

			config.Detach, err = section.Key("detach_on_exec").Bool()
			if err != nil {
				fmt.Fprintf(os.Stderr, "\"%s\" is not a valid boolean value\n", section.Key("detach_on_exec").String())
				fmt.Fprintf(os.Stderr, "The option is ignored and the default value (false) will be used instead\n")
			}
			continue
		}
		handler := Handler{
			MatchExpression: "extension || filter || mime_type || protocol || url_regex",
		}
		prog_cmd_line := section.Key("exec").String()
		handler.Program, err = shlex.Split(prog_cmd_line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot understand %s (from the %s section) as a shell command\n",
				prog_cmd_line, section.Name())
		}
		handler.Extensions = section.Key("extensions").StringsWithShadows(",")
		handler.Filters = section.Key("filter").StringsWithShadows(",")
		handler.MimeTypes = section.Key("mime_type").StringsWithShadows(",")
		handler.Protocols = section.Key("protocol").StringsWithShadows(",")
		handler.UrlRegexs = section.Key("url_regex").StringsWithShadows(",")
		key, _ := section.GetKey("match_expression")
		if key != nil {
			handler.MatchExpression = key.String()
		}
		handler.Name = section.Name()

		config.TypeHandlers = append(config.TypeHandlers, handler)
	}
	// Setup complete, let's goooo !

	var raw_urls []string
	raw_urls = append(raw_urls, flag.Args()...)
	if len(raw_urls) == 0 {
		info, err := os.Stdin.Stat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when trying to stat stdin... good luck: %v\n", err)
			return
		}
		if info.Mode()&os.ModeNamedPipe != 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				flag.Usage()
				return
			}
			raw_urls = strings.Split(string(data), "\n")
			log(LOG_DEBUG, "Data read from stdin: %v\n", raw_urls)
		} else {
			if len(config.ClipboardCmd) == 0 {
				fmt.Fprintf(os.Stderr, "No clipboard command\n")
				return
			}
			log(LOG_DEBUG, "reading from the clipboard\n")
			clipboard := exec.Command(config.ClipboardCmd[0], config.ClipboardCmd[1:]...)
			clipboard.Stdin = nil
			clipboard_output, err := clipboard.StdoutPipe()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting a pipe to the clipboard commands stdout: %v\n", err)
				return
			}
			err = clipboard.Start()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error running the clipboard command: %v\n", err)
				return
			}
			data, err := io.ReadAll(clipboard_output)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting data from the clipboard command: %v\n", err)
				return
			}
			raw_urls = strings.Split(string(data), "\n")
			log(LOG_DEBUG, "Extracted clipboard content: %#v\n", raw_urls)
			clipboard.Wait()
		}
	}

	for _, url := range raw_urls {
		if url == "" {
			// Sometimes, we get an empty url after splitting stdin or the clipboard content
			continue
		}
		log(LOG_DEBUG, "Starting handling of %s\n", url)
		handle_uri(url, config)
	}
}
