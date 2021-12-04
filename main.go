package main

import (
	"fmt"
	_ "mime"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/adrg/xdg"
	"github.com/google/shlex"
	"gopkg.in/ini.v1"
)

type Config struct {
	Browser         []string
	ProgramLauncher []string
	FilterPath      string
	FilterShell     []string
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

var AppName = "url_handler"

func run_filter(filter string, url *url.URL, config *Config, handler Handler) ([]string, string) {
	executable := path.Join(config.FilterPath, filter)
	_, err := os.Stat(executable)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not run the filter \"%s\": %s\n", filter, err)
		return nil, ""
	}
	cmdline := config.FilterShell
	cmdline = append(cmdline, executable)
	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	env := os.Environ()
	env = append(env, fmt.Sprintf("url=%s", url.String())) //XXX this could be slitghly different from str_url
	env = append(env, fmt.Sprintf("protocol=%s", url.Scheme))
	env = append(env, fmt.Sprintf("user=%s", url.User.Username()))
	env = append(env, fmt.Sprintf("host=%s", url.Host))
	env = append(env, fmt.Sprintf("path=%s", url.Path))
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
	fmt.Printf("%#v\n", cmdline)
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running the filter \"%s\": %s\n", filter, err)
	}
	var runner []string
	if cmd.ProcessState.ExitCode() == 0 {
		runner = handler.Program
		buffer := make([]byte, 1204)

		var new_url []byte
		n, err := cmd_stdout.Read(buffer)
		for err == nil && n != 0 {
			new_url = append(new_url, buffer...)
			n, err = cmd_stdout.Read(buffer)
		}
		if len(new_url) != 0 {
			return runner, string(new_url)
		}
		return nil, ""
	}
	return nil, ""
}

func main() {
	config := &Config{
		Browser:      []string{"xdg-open"},
		TypeHandlers: make(map[string]Handler),
	}

	AppName = os.Args[0]

	configFile, err := xdg.SearchConfigFile(path.Join(AppName, "config.ini"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not load config file: %v\n", err)
		return
	}

	conf, err := ini.ShadowLoad(configFile)
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
			if _, err := os.Stat(config.FilterPath); os.IsNotExist(err) {
				fmt.Print(err)
				return
			}
			filter_shell_cmd_line := section.Key("filter_shell").String()
			config.FilterShell, err = shlex.Split(filter_shell_cmd_line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Cannot understand %s as a shell command (from the %s section)\n",
					prog_cmd_line, section.Name())
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

	var str_url string
	//TODO: handle multiple URIs at once
	if len(os.Args) > 1 {
		str_url = os.Args[1]
	} else {
		fmt.Printf("TODO: print usage\n")
		return
	}
	url, err := url.Parse(str_url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in parsing the url %v\n", err)
		return
	}
	parts := strings.Split(url.Path, ".")
	//TODO: get the mimetype if it is a local ressource
	extension := parts[len(parts)-1]

	runner := config.Browser

	for name, handler := range config.TypeHandlers {
		for _, p := range handler.Protocols {
			if p == url.Scheme {
				runner = handler.Program
				break
			}
		}
		//TODO: match the mime type

		for _, ext := range handler.Extensions {
			if ext == extension {
				runner = handler.Program
				break
			}
		}
		for _, reg := range handler.UrlRegexs {
			matched, err := regexp.MatchString(reg, str_url)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error matching the url to a regex: %v\n", err)
			} else if matched {
				runner = handler.Program
				break
			}
		}

		//TODO: support arguments for filters?
		var (
			new_runner []string
			new_url    string
		)

		for _, filter := range handler.Filters {
			if filter == "" {
				// FIXME: this is a wrokaroud for go-ini returning arrays with one empty string
				// if no key is present for some reason
				continue
			}
			new_runner, new_url = run_filter(filter, url, config, handler)

		}
		if len(handler.Filters) == 0 {
			new_runner, new_url = run_filter(name, url, config, handler)
		}
		if new_url != "" {
			//TODO: do something to replace the current url
		}
		if new_runner != nil {
			runner = new_runner
		}

	}

	var cmdline []string
	if len(config.ProgramLauncher) > 0 {
		cmdline = append(cmdline, config.ProgramLauncher...)
	}
	cmdline = append(cmdline, runner...)
	str_url = "\"" + str_url + "\"" // Quoting the url should be enough to not get interferences in the shell
	cmdline = append(cmdline, str_url)
	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error running the command: %v\n", err)
		return
	}
}
