package params

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	// "github.com/sonnt85/goshellbox/client"
	// "github.com/sonnt85/goshellbox/server"
)

// Parameter Command line parameters
type Parameter struct {
	IsServer    bool
	HTTPS       bool
	Host        string
	Port        string
	Username    string
	Password    string
	Command     string
	ContentPath string
	CrtFile     string
	KeyFile     string
	RootCrtFile string
	LocalPort   int
	RemotePort  int
	ProxyPort   string
}

// Init Parameter
func (parms *Parameter) Init(args ...string) (err error) {
	var (
		help, version bool
	)
	u, _ := user.Current()
	username := u.Username
	password := "default123456"
	flagSet := flag.NewFlagSet("goshellbox", flag.ExitOnError)
	flagSet.BoolVar(&help, "h", false, "this help")
	flagSet.BoolVar(&version, "v", false, "show version and exit")
	flagSet.BoolVar(&(parms.IsServer), "s", false, "server mode")
	flagSet.BoolVar(&(parms.HTTPS), "https", false, "enable https")
	flagSet.StringVar(&(parms.Host), "H", "127.0.0.1", "connect to host")
	flagSet.StringVar(&(parms.Port), "P", "2024", "listening port")
	flagSet.StringVar(&(parms.Username), "u", username, "username")
	flagSet.StringVar(&(parms.Password), "p", password, "password")
	flagSet.StringVar(&(parms.Command), "cmd", "", "command cmd or bash")
	flagSet.StringVar(&(parms.ContentPath), "cp", "/shellbox", "content path")
	flagSet.StringVar(&(parms.CrtFile), "C", "", "crt file")
	flagSet.StringVar(&(parms.KeyFile), "K", "", "key file")
	flagSet.StringVar(&(parms.RootCrtFile), "RC", "", "root crt file")
	flagSet.IntVar(&(parms.LocalPort), "LP", 2222, "ssh port for proxy")
	flagSet.IntVar(&(parms.RemotePort), "RP", 22, "ssh port for proxy")
	flagSet.StringVar(&(parms.ProxyPort), "PP", "", "ssh port for proxy")

	if len(args) == 0 {
		args = os.Args[1:]
	}
	err = flagSet.Parse(args)
	if u := os.Getenv("SB_USERNAME"); len(u) != 0 {
		parms.Username = u
	}
	if p := os.Getenv("SB_PASSWORD"); len(p) != 0 {
		parms.Password = p
	}
	if err != nil {
		// fmt.Println(err)
		return
		// os.Exit(1)
	}
	if help {
		printUsage()
		flagSet.PrintDefaults()
		// os.Exit(1)
		return
	} else if version {
		printVersion()
		// os.Exit(1)
		return
	} else {
		// fmt.Printf("%+v\n", parms)
		return parms.organize()
	}
}

// Run start server or client
// func (parms *Parameter) Run() {
// if parms.Server {
// 	server.Run(parms)
// } else if parms.Client {
// 	c := new(client.ShellBoxClient)
// 	c.Init(parms.HTTPS, parms.CrtFile, parms.KeyFile, parms.RootCrtFile)
// 	c.Run(parms)
// }
// }

// organize command line parameters
func (parms *Parameter) organize() (err error) {
	if parms.IsServer && parms.HTTPS && (parms.CrtFile == "" || parms.KeyFile == "") {
		// println("the crt file and key file are required in server mode.")
		err = fmt.Errorf("the crt file and key file are required in server mode")
		return
		// os.Exit(1)
	}
	_, err = strconv.Atoi(parms.Port)
	if err != nil {
		// println("Port must be an int, not" + parms.Port)
		return fmt.Errorf("Port must be an int, not" + parms.Port)
		// os.Exit(1)
	}
	parms.Command = strings.Trim(parms.Command, " ")
	if parms.Command == "" {
		parms.Command = defaultCommand()
	}
	if parms.Username == "" {
		parms.Username = getInput("Username")
	}
	if parms.Password == "" {
		parms.Password = getInput("Password")
	}
	parms.ContentPath = strings.Trim(parms.ContentPath, " ")
	if len(parms.ContentPath) > 0 {
		if parms.ContentPath[0] != '/' {
			// println("ContentPath must start with /, not", parms.ContentPath)
			return fmt.Errorf("ContentPath string must start with / [%s]", parms.ContentPath)
			// os.Exit(1)
		}
		if parms.ContentPath[len(parms.ContentPath)-1] == '/' {
			// println("ContentPath cannot end with /, not", parms.ContentPath)
			return fmt.Errorf("ContentPath must not end with / [%s]", parms.ContentPath)
			// os.Exit(1)
		}
	}
	return nil
}

func printUsage() {
	println(`Usage:
  goshellbox [-s server mode] [-c client mode]  [-P port] [-u username] [-p password] [-cmd command]

Example:
  goshellbox -s -H 192.168.1.1 -P 2024 -u admin -p admin -https -cmd bash
  goshellbox -c -H 192.168.1.1 -P 2024 -u admin -p admin -https

Options:`)
}

func printVersion() {
	// println("goshellbox server version:", server.Version)
	// println("goshellbox client version:", client.Version)
}

// defaultCommand Get the default shell
func defaultCommand() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	for _, cmd := range []string{"bash", "sh"} {
		if c, e := exec.LookPath(cmd); e == nil {
			return c
		}
	}
	return "bash"
}

// getInput Get input from the command line
func getInput(key string) string {
	pwd := ""
	fmt.Print("Enter " + key + ": ")
	fmt.Scanln(&pwd)
	if pwd == "" {
		return getInput(key)
	}
	return pwd
}

// organizeOsArgs Organize os.Args
// The parameters -u, -p are allowed to be empty
func organizeOsArgs(osArgs []string) []string {
	args := make([]string, 0)
	for i, arg := range osArgs {
		args = append(args, arg)
		if arg == "-u" {
			if len(osArgs) <= i+1 {
				args = append(args, "")
				return args
			}
			u := osArgs[i+1]
			if strings.HasPrefix(u, "-") {
				u = strings.TrimLeft(u, "-")
				if flag.CommandLine.Lookup(u) != nil {
					args = append(args, "")
				}
			}
		}
		if arg == "-p" {
			if len(osArgs) <= i+1 {
				args = append(args, "")
				return args
			}
			p := osArgs[i+1]
			if strings.HasPrefix(p, "-") {
				p = strings.TrimLeft(p, "-")
				if flag.CommandLine.Lookup(p) != nil {
					args = append(args, "")
				}
			}
		}
	}
	return args
}
