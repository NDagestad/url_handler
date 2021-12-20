package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

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
	TypeHandlers    map[string]Handler
}

type Handler struct {
	Program    []string
	Extensions []string
	Filters    []string
	MimeTypes  []string
	Protocols  []string
	UrlRegexs  []string
}

var (
	AppName string
	debug   *bool
)

func log(format string, args ...interface{}) {
	if *debug {
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
			mime = resp.Header["Content-Type"][0]
		}
	case "file":
		fallthrough
	case "":
		mtype, err := mimetype.DetectFile(ressource.Path)
		if err == nil {
			mime = mtype.String()
		} else {
			log("Error getting mime type\n")
		}
	}
	return mime, err
}

func run_filter(section_name string, filter string, url *URL, config *Config, handler Handler) ([]string, string) {
	var executable string
	if config.FilterPath != "" {
		executable = filepath.Join(config.FilterPath, filter)
		_, err := os.Stat(executable)
		if err != nil {
			if section_name != filter {
				fmt.Fprintf(os.Stderr, "Could not run the filter \"%s\": %s\n", filter, err)
			}
			return nil, ""
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
	env = append(env, fmt.Sprintf("section=%s", section_name))
	cmd_stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get a pipe to the filters stdin: %v\n", err)
	}
	pwd, available := url.User.Password()
	if available && err == nil {
		cmd_stdin.Write([]byte(pwd))
	}
	cmd_stdin.Close()
	cmd_stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get a pipe to the filters stdout: %v\n", err)
	}
	cmd.Env = env
	log("Running %#v\n", cmdline)
	err = cmd.Run()
	if err != nil && cmd.ProcessState.ExitCode() != 1 {
		fmt.Fprintf(os.Stderr, "Error running the filter \"%s\": %s\n", filter, err)
	} else if cmd.ProcessState.ExitCode() == 0 {
		new_url, err := ioutil.ReadAll(cmd_stdout)
		if err != nil {
			log("Error reading stdout from the filter (%s) output: %v\n", filter, err)
		}
		return handler.Program, string(new_url)
	} else if cmd.ProcessState.ExitCode() == 1 {
		log("Filter (%s) returned 1\n", filter)
		return nil, ""
	}
	return nil, ""
}

func handle_uri(raw_url string, config *Config) {

	url, err := Parse(raw_url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in parsing the url %v\n", err)
		return
	}

	parts := strings.Split(url.Path, ".")
	extension := parts[len(parts)-1]
	mime_type, err := get_mime_type(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get mime type for %s: %v\n", url.String(), err)
	}

	runner := config.Browser

handler:
	for name, handler := range config.TypeHandlers {
		log("Checking matchs for %s\n", name)
		for _, p := range handler.Protocols {
			if p == "" {
				// Issue reported and solution proposed
				// FIXME: workaround for go-ini giving us an array with an empty string instead
				//of an empty array
				break
			}
			if p == url.Scheme {
				log("Matched with the protocol for %#v\n", p)
				runner = handler.Program
				break handler
			}
		}
		for _, mime := range handler.MimeTypes {
			if mime == "" {
				// FIXME: workaround for go-ini giving us an array with an empty string instead
				//of an empty array
				break
			}
			matched, err := regexp.MatchString(mime, mime_type)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s is not a valide regex, ignored...\n", mime)
				continue
			}
			if matched {
				log("Matched with the mime-type for %#v\n", mime)
				runner = handler.Program
				break handler
			}
		}

		for _, ext := range handler.Extensions {
			if ext == "" {
				// FIXME: workaround for go-ini giving us an array with an empty string instead
				//of an empty array
				break
			}
			if ext == extension {
				log("Matched with the extension for %#v\n", ext)
				runner = handler.Program
				break handler
			}
		}
		for _, reg := range handler.UrlRegexs {
			if reg == "" {
				// FIXME: workaround for go-ini giving us an array with an empty string instead
				//of an empty array
				break
			}
			matched, err := regexp.MatchString(reg, raw_url)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error matching the url to a regex: %v\n", err)
			} else if matched {
				log("Matched with a regex for %#v\n", reg)
				runner = handler.Program
				break handler
			}
		}

		//TODO: support arguments for filters?
		var (
			new_runner []string
			new_url    string
		)

		for _, filter := range handler.Filters {
			if filter == "" {
				// FIXME: workaround for go-ini giving us an array with an empty string instead
				//of an empty array
				break
			}
			new_runner, new_url = run_filter(name, filter, url, config, handler)
			if new_url != "" {
				//TODO: do something to replace the current url
				log("(%s):A filter gave us a new url: %s\n", filter, new_url)
				//break handler
			}
			if new_runner != nil {
				runner = new_runner
				log("Matched because of a filter: %s\n", filter)
				break handler
			}
		}
		//if len(handler.Filters) == 0 {
		if handler.Filters[0] == "" { //FIXME: workaround for the above go-ini bug
			new_runner, new_url = run_filter(name, name, url, config, handler)
			if new_url != "" {
				//TODO: do something to replace the current url
				log("(%s)The default filter gave us a new url: %s\n", name, new_url)
				//break handler
			}
			if new_runner != nil {
				runner = new_runner
				log("Matched because of the default filter: %s\n", name)
				break handler
			}
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

	log("Handling the url with: %#v\n", cmdline)

	// TODO make the program fork to not hang waiting for the cmd to exit
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error running the command: %v\n", err)
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
	config := &Config{
		Browser:      []string{"xdg-open"},
		TypeHandlers: make(map[string]Handler),
	}

	debug = flag.Bool("debug", false, "Enable debug output")
	flag.Parse()

	AppName = filepath.Base(os.Args[0])
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
			continue
		}
		_, exists := config.TypeHandlers[section.Name()]
		if !exists {
			handler := Handler{}
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
			config.TypeHandlers[section.Name()] = handler

		} else {
			fmt.Fprintf(os.Stderr, "There is more than one %s section, section names should be unique, exiting\n",
				section.Name())
			return
		}
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
			data, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				flag.Usage()
				return
			}
			raw_urls = strings.Split(string(data), "\n")
			log("Data read from stdin: %v\n", raw_urls)
		} else {
			if len(config.ClipboardCmd) == 0 {
				fmt.Fprintf(os.Stderr, "No clipboard command\n")
				return
			}
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
			data, err := ioutil.ReadAll(clipboard_output)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting data from the clipboard command: %v\n", err)
				return
			}
			raw_urls = strings.Split(string(data), "\n")
			log("Extracted clipboard content: %#v\n", raw_urls)
			clipboard.Wait()
		}
	}

	for _, url := range raw_urls {
		if url == "" {
			// Sometimes, we get an empty url after splitting stdin or the clipboard content
			continue
		}
		log("Starting handling of %s\n", url)
		handle_uri(url, config)
	}
}
